# TODO(pphaneuf): Make this be good.

.PHONY: configure-ct openssl protobuf libevent libevhtp gflags glog ldns sqlite3 leveldb json-c tcmalloc

all: configure-ct

tcmalloc:
	$(MAKE) -C gflags -f ../certificate-transparency/build/Makefile.tcmalloc

openssl:
	make -C openssl -f `pwd`/certificate-transparency/build/Makefile.openssl INSTALL=$(INSTALL)

protobuf:
	certificate-transparency/build/rebuild_protobuf

libevent:
	certificate-transparency/build/rebuild_libevent

libevhtp:
	certificate-transparency/build/rebuild_libevhtp

gflags:
	$(MAKE) -C gflags -f ../certificate-transparency/build/Makefile.gflags

glog:
	$(MAKE) -C glog -f ../certificate-transparency/build/Makefile.glog

ldns:
	$(MAKE) -C ldns -f ../certificate-transparency/build/Makefile.ldns

sqlite3:
	$(MAKE) -C sqlite3 -f ../certificate-transparency/build/Makefile.sqlite3

leveldb:
	$(MAKE) -C leveldb -f ../certificate-transparency/build/Makefile.leveldb

json-c:
	certificate-transparency/build/rebuild_json-c

configure-ct: tcmalloc openssl protobuf libevent libevhtp gflags glog ldns sqlite3 leveldb json-c
	certificate-transparency/build/configure-ct
