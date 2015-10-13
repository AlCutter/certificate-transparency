#ifndef CERT_TRANS_SERVER_HANDLER_H_
#define CERT_TRANS_SERVER_HANDLER_H_

#include <algorithm>
#include <event2/buffer.h>
#include <event2/http.h>
#include <functional>
#include <gflags/gflags.h>
#include <glog/logging.h>
#include <memory>
#include <mutex>
#include <stdint.h>
#include <stdlib.h>
#include <string>
#include <utility>
#include <vector>

#include "log/cert_checker.h"
#include "log/cluster_state_controller.h"
#include "log/frontend.h"
#include "log/log_lookup.h"
#include "log/logged_entry.h"
#include "monitoring/monitoring.h"
#include "monitoring/latency.h"
#include "server/json_output.h"
#include "server/proxy.h"
#include "util/json_wrapper.h"
#include "util/libevent_wrapper.h"
#include "util/sync_task.h"
#include "util/task.h"
#include "util/thread_pool.h"

DECLARE_int32(max_leaf_entries_per_response);
DECLARE_int32(staleness_check_delay_secs);

namespace cert_trans {

void StatsHandlerInterceptor(const std::string& path,
                             const libevent::HttpServer::HandlerCallback& cb,
                             evhttp_request* req);

template <class Logged>
class HttpHandler {
 public:
  // Does not take ownership of its parameters, which must outlive
  // this instance. The "frontend" parameter can be NULL, in which
  // case this server will not accept "add-chain" and "add-pre-chain"
  // requests.
  HttpHandler(JsonOutput* json_output, LogLookup<Logged>* log_lookup,
              const ReadOnlyDatabase<Logged>* db,
              const ClusterStateController<Logged>* controller, Proxy* proxy,
              ThreadPool* pool, libevent::Base* event_base);
  ~HttpHandler();

  void Add(libevent::HttpServer* server);

 protected:
  virtual void AddHandlers(libevent::HttpServer* server) = 0;

  void AddEntryReply(evhttp_request* req, const util::Status& add_status,
                     const ct::SignedCertificateTimestamp& sct) const;

  void ProxyInterceptor(
      const libevent::HttpServer::HandlerCallback& local_handler,
      evhttp_request* request);

  void AddProxyWrappedHandler(
      libevent::HttpServer* server, const std::string& path,
      const libevent::HttpServer::HandlerCallback& local_handler);

  void GetEntries(evhttp_request* req) const;
  void GetProof(evhttp_request* req) const;
  void GetSTH(evhttp_request* req) const;
  void GetConsistency(evhttp_request* req) const;

  void BlockingGetEntries(evhttp_request* req, int64_t start, int64_t end,
                          bool include_scts) const;

  bool IsNodeStale() const;
  void UpdateNodeStaleness();

  JsonOutput* const output_;
  LogLookup<Logged>* const log_lookup_;
  const ReadOnlyDatabase<Logged>* const db_;
  const ClusterStateController<Logged>* const controller_;
  Proxy* const proxy_;
  ThreadPool* const pool_;
  libevent::Base* const event_base_;

  util::SyncTask task_;
  mutable std::mutex mutex_;
  bool node_is_stale_;

  DISALLOW_COPY_AND_ASSIGN(HttpHandler);
};


template <class Logged>
HttpHandler<Logged>::HttpHandler(
    JsonOutput* output, LogLookup<Logged>* log_lookup,
    const ReadOnlyDatabase<Logged>* db,
    const ClusterStateController<Logged>* controller, Proxy* proxy,
    ThreadPool* pool, libevent::Base* event_base)
    : output_(CHECK_NOTNULL(output)),
      log_lookup_(CHECK_NOTNULL(log_lookup)),
      db_(CHECK_NOTNULL(db)),
      controller_(CHECK_NOTNULL(controller)),
      proxy_(CHECK_NOTNULL(proxy)),
      pool_(CHECK_NOTNULL(pool)),
      event_base_(CHECK_NOTNULL(event_base)),
      task_(pool_),
      node_is_stale_(controller_->NodeIsStale()) {
  event_base_->Delay(std::chrono::seconds(FLAGS_staleness_check_delay_secs),
                     task_.task()->AddChild(
                         std::bind(&HttpHandler::UpdateNodeStaleness, this)));
}


template <class Logged>
HttpHandler<Logged>::~HttpHandler() {
  task_.task()->Return();
  task_.Wait();
}


template <class Logged>
void HttpHandler<Logged>::AddEntryReply(
    evhttp_request* req, const util::Status& add_status,
    const ct::SignedCertificateTimestamp& sct) const {
  if (!add_status.ok() &&
      add_status.CanonicalCode() != util::error::ALREADY_EXISTS) {
    VLOG(1) << "error adding chain: " << add_status;
    const int response_code(add_status.CanonicalCode() ==
                                    util::error::RESOURCE_EXHAUSTED
                                ? HTTP_SERVUNAVAIL
                                : HTTP_BADREQUEST);
    return output_->SendError(req, response_code, add_status.error_message());
  }

  JsonObject json_reply;
  json_reply.Add("sct_version", static_cast<int64_t>(0));
  json_reply.AddBase64("id", sct.id().key_id());
  json_reply.Add("timestamp", sct.timestamp());
  json_reply.Add("extensions", "");
  json_reply.Add("signature", sct.signature());

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}

template <class Logged>
void HttpHandler<Logged>::ProxyInterceptor(
    const libevent::HttpServer::HandlerCallback& local_handler,
    evhttp_request* request) {
  VLOG(2) << "Running proxy interceptor...";
  // TODO(alcutter): We can be a bit smarter about when to proxy off
  // the request - being stale wrt to the current serving STH doesn't
  // automatically mean we're unable to answer this request.
  if (IsNodeStale()) {
    // Can't do this on the libevent thread since it can block on the lock in
    // ClusterStatusController::GetFreshNodes().
    pool_->Add(std::bind(&Proxy::ProxyRequest, proxy_, request));
  } else {
    local_handler(request);
  }
}


template <class Logged>
void HttpHandler<Logged>::AddProxyWrappedHandler(
    libevent::HttpServer* server, const std::string& path,
    const libevent::HttpServer::HandlerCallback& local_handler) {
  const libevent::HttpServer::HandlerCallback stats_handler(
      bind(&StatsHandlerInterceptor, path, local_handler,
           std::placeholders::_1));
  CHECK(server->AddHandler(path, bind(&HttpHandler::ProxyInterceptor, this,
                                      stats_handler, std::placeholders::_1)));
}


template <class Logged>
void HttpHandler<Logged>::Add(libevent::HttpServer* server) {
  CHECK_NOTNULL(server);
  // TODO(pphaneuf): An optional prefix might be nice?
  // TODO(pphaneuf): Find out which methods are CPU intensive enough
  // that they should be spun off to the thread pool.
  AddProxyWrappedHandler(server, "/ct/v1/get-entries",
                         bind(&HttpHandler::GetEntries, this,
                              std::placeholders::_1));
  AddProxyWrappedHandler(server, "/ct/v1/get-proof-by-hash",
                         bind(&HttpHandler::GetProof, this,
                              std::placeholders::_1));
  AddProxyWrappedHandler(server, "/ct/v1/get-sth",
                         bind(&HttpHandler::GetSTH, this,
                              std::placeholders::_1));
  AddProxyWrappedHandler(server, "/ct/v1/get-sth-consistency",
                         bind(&HttpHandler::GetConsistency, this,
                              std::placeholders::_1));

  AddHandlers(server);
}


template <class Logged>
void HttpHandler<Logged>::GetEntries(evhttp_request* req) const {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_GET) {
    return output_->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
  }

  const libevent::QueryParams query(libevent::ParseQuery(req));

  const int64_t start(libevent::GetIntParam(query, "start"));
  if (start < 0) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"start\" parameter.");
  }

  int64_t end(libevent::GetIntParam(query, "end"));
  if (end < start) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"end\" parameter.");
  }

  // Limit the number of entries returned in a single request.
  end = std::min(end, start + FLAGS_max_leaf_entries_per_response);

  // Sekrit parameter to indicate that SCTs should be included too.
  // This is non-standard, and is only used internally by other log nodes when
  // "following" nodes with more data.
  const bool include_scts(libevent::GetBoolParam(query, "include_scts"));

  BlockingGetEntries(req, start, end, include_scts);
}


template <class Logged>
void HttpHandler<Logged>::GetProof(evhttp_request* req) const {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_GET) {
    return output_->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
  }

  const libevent::QueryParams query(libevent::ParseQuery(req));

  std::string b64_hash;
  if (!libevent::GetParam(query, "hash", &b64_hash)) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"hash\" parameter.");
  }

  const std::string hash(util::FromBase64(b64_hash.c_str()));
  if (hash.empty()) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Invalid \"hash\" parameter.");
  }

  const int64_t tree_size(libevent::GetIntParam(query, "tree_size"));
  if (tree_size < 0 ||
      static_cast<int64_t>(tree_size) > log_lookup_->GetSTH().tree_size()) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"tree_size\" parameter.");
  }

  ct::ShortMerkleAuditProof proof;
  if (log_lookup_->AuditProof(hash, tree_size, &proof) !=
      LogLookup<Logged>::OK) {
    return output_->SendError(req, HTTP_BADREQUEST, "Couldn't find hash.");
  }

  JsonArray json_audit;
  for (int i = 0; i < proof.path_node_size(); ++i) {
    json_audit.AddBase64(proof.path_node(i));
  }

  JsonObject json_reply;
  json_reply.Add("leaf_index", proof.leaf_index());
  json_reply.Add("audit_path", json_audit);

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}


template <class Logged>
void HttpHandler<Logged>::GetSTH(evhttp_request* req) const {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_GET) {
    return output_->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
  }

  const ct::SignedTreeHead& sth(log_lookup_->GetSTH());

  VLOG(2) << "SignedTreeHead:\n" << sth.DebugString();

  JsonObject json_reply;
  json_reply.Add("tree_size", sth.tree_size());
  json_reply.Add("timestamp", sth.timestamp());
  json_reply.AddBase64("sha256_root_hash", sth.sha256_root_hash());
  json_reply.Add("tree_head_signature", sth.signature());

  VLOG(2) << "GetSTH:\n" << json_reply.DebugString();

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}


template <class Logged>
void HttpHandler<Logged>::GetConsistency(evhttp_request* req) const {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_GET) {
    return output_->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
  }

  const libevent::QueryParams query(libevent::ParseQuery(req));

  const int64_t first(libevent::GetIntParam(query, "first"));
  if (first < 0) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"first\" parameter.");
  }

  const int64_t second(libevent::GetIntParam(query, "second"));
  if (second < first) {
    return output_->SendError(req, HTTP_BADREQUEST,
                              "Missing or invalid \"second\" parameter.");
  }

  const std::vector<std::string> consistency(
      log_lookup_->ConsistencyProof(first, second));
  JsonArray json_cons;
  for (std::vector<std::string>::const_iterator it = consistency.begin();
       it != consistency.end(); ++it) {
    json_cons.AddBase64(*it);
  }

  JsonObject json_reply;
  json_reply.Add("consistency", json_cons);

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}


template <class Logged>
void HttpHandler<Logged>::BlockingGetEntries(evhttp_request* req,
                                             int64_t start, int64_t end,
                                             bool include_scts) const {
  JsonArray json_entries;
  auto it(db_->ScanEntries(start));
  for (int64_t i = start; i <= end; ++i) {
    Logged entry;

    if (!it->GetNextEntry(&entry) || entry.sequence_number() != i) {
      break;
    }

    std::string leaf_input;
    std::string extra_data;
    std::string sct_data;
    if (!entry.SerializeForLeaf(&leaf_input) ||
        !entry.SerializeExtraData(&extra_data) ||
        (include_scts &&
         Serializer::SerializeSCT(entry.sct(), &sct_data) != Serializer::OK)) {
      LOG(WARNING) << "Failed to serialize entry @ " << i << ":\n"
                   << entry.DebugString();
      return output_->SendError(req, HTTP_INTERNAL, "Serialization failed.");
    }

    JsonObject json_entry;
    json_entry.AddBase64("leaf_input", leaf_input);
    json_entry.AddBase64("extra_data", extra_data);

    if (include_scts) {
      // This is non-standard, and currently only used by other SuperDuper log
      // nodes when "following" to fetch data from each other:
      json_entry.AddBase64("sct", sct_data);
    }

    json_entries.Add(&json_entry);
  }

  if (json_entries.Length() < 1) {
    return output_->SendError(req, HTTP_BADREQUEST, "Entry not found.");
  }

  JsonObject json_reply;
  json_reply.Add("entries", json_entries);

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}


template <class Logged>
bool HttpHandler<Logged>::IsNodeStale() const {
  std::lock_guard<std::mutex> lock(mutex_);
  return node_is_stale_;
}


template <class Logged>
void HttpHandler<Logged>::UpdateNodeStaleness() {
  if (!task_.task()->IsActive()) {
    // We're shutting down, just return.
    return;
  }

  const bool node_is_stale(controller_->NodeIsStale());
  {
    std::lock_guard<std::mutex> lock(mutex_);
    node_is_stale_ = node_is_stale;
  }

  event_base_->Delay(std::chrono::seconds(FLAGS_staleness_check_delay_secs),
                     task_.task()->AddChild(
                         std::bind(&HttpHandler::UpdateNodeStaleness, this)));
}

}  // namespace cert_trans

#endif  // CERT_TRANS_SERVER_HANDLER_H_
