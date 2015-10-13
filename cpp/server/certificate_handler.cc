#include "server/certificate_handler.h"

namespace cert_trans {

using ct::SignedCertificateTimestamp;
using std::make_shared;
using std::multimap;
using std::placeholders::_1;
using std::shared_ptr;
using std::string;
using std::unique_ptr;


namespace {


bool ExtractChain(JsonOutput* output, evhttp_request* req, CertChain* chain) {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_POST) {
    output->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
    return false;
  }

  // TODO(pphaneuf): Should we check that Content-Type says
  // "application/json", as recommended by RFC4627?
  JsonObject json_body(evhttp_request_get_input_buffer(req));
  if (!json_body.Ok() || !json_body.IsType(json_type_object)) {
    output->SendError(req, HTTP_BADREQUEST, "Unable to parse provided JSON.");
    return false;
  }

  JsonArray json_chain(json_body, "chain");
  if (!json_chain.Ok()) {
    output->SendError(req, HTTP_BADREQUEST, "Unable to parse provided JSON.");
    return false;
  }

  VLOG(2) << "ExtractChain chain:\n" << json_chain.DebugString();

  for (int i = 0; i < json_chain.Length(); ++i) {
    JsonString json_cert(json_chain, i);
    if (!json_cert.Ok()) {
      output->SendError(req, HTTP_BADREQUEST,
                        "Unable to parse provided JSON.");
      return false;
    }

    unique_ptr<Cert> cert(new Cert);
    cert->LoadFromDerString(json_cert.FromBase64());
    if (!cert->IsLoaded()) {
      output->SendError(req, HTTP_BADREQUEST,
                        "Unable to parse provided chain.");
      return false;
    }

    chain->AddCert(cert.release());
  }

  return true;
}


}  // namespace


CertificateHttpHandler::CertificateHttpHandler(
    JsonOutput* json_output, LogLookup<LoggedEntry>* log_lookup,
    const ReadOnlyDatabase<LoggedEntry>* db,
    const ClusterStateController<LoggedEntry>* controller,
    const CertChecker* cert_checker, Frontend* frontend, Proxy* proxy,
    ThreadPool* pool, libevent::Base* event_base)
    : HttpHandler<LoggedEntry>(json_output, log_lookup, db, controller, proxy,
                               pool, event_base),
      cert_checker_(cert_checker),
      frontend_(frontend) {
}


void CertificateHttpHandler::AddHandlers(libevent::HttpServer* server) {
  // TODO(alcutter): Support this for mirrors too
  if (cert_checker_) {
    // Don't really need to proxy this one, but may as well just to keep
    // everything tidy:
    AddProxyWrappedHandler(server, "/ct/v1/get-roots",
                           bind(&CertificateHttpHandler::GetRoots, this, _1));
  }
  if (frontend_) {
    // Proxy the add-* calls too, technically we could serve them, but a
    // more up-to-date node will have a better chance of handling dupes
    // correctly, rather than bloating the tree.
    AddProxyWrappedHandler(server, "/ct/v1/add-chain",
                           bind(&CertificateHttpHandler::AddChain, this, _1));
    AddProxyWrappedHandler(server, "/ct/v1/add-pre-chain",
                           bind(&CertificateHttpHandler::AddPreChain, this,
                                _1));
  }
}


void CertificateHttpHandler::GetRoots(evhttp_request* req) const {
  if (evhttp_request_get_command(req) != EVHTTP_REQ_GET) {
    return output_->SendError(req, HTTP_BADMETHOD, "Method not allowed.");
  }

  JsonArray roots;
  multimap<string, const Cert*>::const_iterator it;
  for (it = cert_checker_->GetTrustedCertificates().begin();
       it != cert_checker_->GetTrustedCertificates().end(); ++it) {
    string cert;
    if (it->second->DerEncoding(&cert) != util::Status::OK) {
      LOG(ERROR) << "Cert encoding failed";
      return output_->SendError(req, HTTP_INTERNAL, "Serialisation failed.");
    }
    roots.AddBase64(cert);
  }

  JsonObject json_reply;
  json_reply.Add("certificates", roots);

  output_->SendJsonReply(req, HTTP_OK, json_reply);
}


void CertificateHttpHandler::AddChain(evhttp_request* req) {
  const shared_ptr<CertChain> chain(make_shared<CertChain>());
  if (!ExtractChain(output_, req, chain.get())) {
    return;
  }

  pool_->Add(
      bind(&CertificateHttpHandler::BlockingAddChain, this, req, chain));
}


void CertificateHttpHandler::AddPreChain(evhttp_request* req) {
  const shared_ptr<PreCertChain> chain(make_shared<PreCertChain>());
  if (!ExtractChain(output_, req, chain.get())) {
    return;
  }

  pool_->Add(
      bind(&CertificateHttpHandler::BlockingAddPreChain, this, req, chain));
}


void CertificateHttpHandler::BlockingAddChain(
    evhttp_request* req, const shared_ptr<CertChain>& chain) const {
  SignedCertificateTimestamp sct;

  AddEntryReply(req, CHECK_NOTNULL(frontend_)
                         ->QueueX509Entry(CHECK_NOTNULL(chain.get()), &sct),
                sct);
}


void CertificateHttpHandler::BlockingAddPreChain(
    evhttp_request* req, const shared_ptr<PreCertChain>& chain) const {
  SignedCertificateTimestamp sct;

  AddEntryReply(req, CHECK_NOTNULL(frontend_)
                         ->QueuePreCertEntry(CHECK_NOTNULL(chain.get()), &sct),
                sct);
}


}  // namespace cert_trans
