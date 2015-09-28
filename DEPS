deps = {
     "googlemock": "https://github.com/google/googlemock.git@release-1.7.0",
     "googlemock/gtest": "https://github.com/google/googletest.git@release-1.7.0",
     "openssl": "https://github.com/benlaurie/openssl.git@fd5d2ba5e09f86e9ccf797dddd2d09ac8e197e35", # 1.0.2-freebsd
     "protobuf/gtest": "https://github.com/google/googletest.git@release-1.5.0",
     "protobuf": "https://github.com/google/protobuf.git@v2.6.1",
     "libevent": "https://github.com/libevent/libevent.git@release-2.0.22-stable",
     "libevhtp": "https://github.com/ellzey/libevhtp.git@ba4c44eed1fb7a5cf8e4deb236af4f7675cc72d5",
     "gflags": "https://github.com/gflags/gflags.git@v2.1.2",
     "glog": "https://github.com/google/glog.git@v0.3.4",
     "ldns": "git://git.nlnetlabs.nl/ldns@release-1.6.17",
     # Randomly chosen github mirror
     "sqlite3-export": "http://repo.or.cz/sqlite-export.git",
     "sqlite3": "http://repo.or.cz/sqlite.git@version-3.8.10.1",
     "leveldb": "https://github.com/google/leveldb.git@v1.18",
     "json-c": "https://github.com/json-c/json-c.git@json-c-0.12-20140410",
}

# Can't use deps_os for this because it doesn't know about freebsd :/
deps_overrides = {
  "freebsd10": {
     "protobuf/gtest": "https://github.com/benlaurie/googletest.git@1.5.0-fix",
     "protobuf": "https://github.com/benlaurie/protobuf.git@2.6.1-fix",
     "glog": "https://github.com/benlaurie/glog.git@0.3.4-fix",
     "ldns": "https://github.com/benlaurie/ldns.git@1.6.17-fix",
  },
  "darwin": {
     "ldns": "https://github.com/benlaurie/ldns.git@1.6.17-fix",
  }
}

import os
import sys

print "Host platform is %s" % sys.platform
if sys.platform in deps_overrides:
  print "Have %d overrides for platform" % len(deps_overrides[sys.platform])
  deps.update(deps_overrides[sys.platform])

here = os.getcwd()
install = os.path.join(here, "install")


hooks = [
    {
        "name": "openssl",
        "pattern": "^openssl/",
        "action": [ "make", "-C", "openssl", "-f", os.path.join(here, "certificate-transparency/build/Makefile.openssl"), "INSTALL=" + install ],
    },
    {
        "name": "protobuf",
        "pattern": "^protobuf/",
        "action": [ "certificate-transparency/build/rebuild_protobuf" ],
    },
    {
        "name": "libevent",
        "pattern": "^libevent/",
        "action": [ "certificate-transparency/build/rebuild_libevent" ],
    },
    {
        "name": "libevhtp",
        "pattern": "^libevhtp/",
        "action": [ "certificate-transparency/build/rebuild_libevhtp" ],
    },
    {
        "name": "gflags",
        "pattern": "^gflags/",
        "action": [ "make", "-C", "gflags", "-f", os.path.join(here, "certificate-transparency/build/Makefile.gflags") ],
    },
    {
        "name": "glog",
        "pattern": "^glog/",
        "action": [ "make", "-C", "glog", "-f",  os.path.join(here, "certificate-transparency/build/Makefile.glog") ],
    },
    {
        "name": "ldns",
        "pattern": "^ldns/",
        "action": [ "make", "-C", "ldns", "-f",  os.path.join(here, "certificate-transparency/build/Makefile.ldns") ],
    },
    {
        "name": "sqlite3",
        "pattern": "^sqlite3/",
        "action": [ "make", "-C", "sqlite3", "-f",  os.path.join(here, "certificate-transparency/build/Makefile.sqlite3") ],
    },
    {
        "name": "leveldb",
        "pattern": "^leveldb/",
        "action": [ "make", "-C", "leveldb", "-f",  os.path.join(here, "certificate-transparency/build/Makefile.leveldb") ],
    },
    {
        "name": "json-c",
        "pattern": "^json-c/",
        "action": [ "certificate-transparency/build/rebuild_json-c" ],
    },
    # Do this last
    {
        "name": "ct",
        "pattern": ".",
        "action": [ "certificate-transparency/build/rebuild" ],
    }
]
