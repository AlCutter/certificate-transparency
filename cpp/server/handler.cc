#include <gflags/gflags.h>

#include "server/handler.h"

DEFINE_int32(max_leaf_entries_per_response, 1000,
             "maximum number of entries to put in the response of a "
             "get-entries request");
DEFINE_int32(staleness_check_delay_secs, 5,
             "number of seconds between node staleness checks");


namespace cert_trans {

using std::string;

namespace {

static Latency<std::chrono::milliseconds, std::string>
    http_server_request_latency_ms(
        "total_http_server_request_latency_ms", "path",
        "Total request latency in ms broken down by path");

}  // namespace


void StatsHandlerInterceptor(const std::string& path,
                             const libevent::HttpServer::HandlerCallback& cb,
                             evhttp_request* req) {
  ScopedLatency total_http_server_request_latency(
      http_server_request_latency_ms.GetScopedLatency(path));

  cb(req);
}


}  // namespace cert_trans
