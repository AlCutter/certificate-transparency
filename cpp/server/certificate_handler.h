#ifndef CERT_TRANS_SERVER_CERTIFICATE_HANDLER_H_
#define CERT_TRANS_SERVER_CERTIFICATE_HANDLER_H_

#include "log/logged_entry.h"
#include "server/handler.h"

namespace cert_trans {


class CertificateHttpHandler : public HttpHandler<LoggedEntry> {
 public:
  CertificateHttpHandler(JsonOutput* json_output,
                         LogLookup<LoggedEntry>* log_lookup,
                         const ReadOnlyDatabase<LoggedEntry>* db,
                         const ClusterStateController<LoggedEntry>* controller,
                         const CertChecker* cert_checker, Frontend* frontend,
                         Proxy* proxy, ThreadPool* pool,
                         libevent::Base* event_base);

  ~CertificateHttpHandler() = default;

 protected:
  void AddHandlers(libevent::HttpServer* server);

 private:
  const CertChecker* const cert_checker_;
  Frontend* const frontend_;

  void GetEntries(evhttp_request* req) const;
  void GetRoots(evhttp_request* req) const;
  void AddChain(evhttp_request* req);
  void AddPreChain(evhttp_request* req);

  void BlockingAddChain(evhttp_request* req,
                        const std::shared_ptr<CertChain>& chain) const;
  void BlockingAddPreChain(evhttp_request* req,
                           const std::shared_ptr<PreCertChain>& chain) const;

  DISALLOW_COPY_AND_ASSIGN(CertificateHttpHandler);
};


}  // namespace cert_trans


#endif  // CERT_TRANS_SERVER_CERTIFICATE_HANDLER_H_
