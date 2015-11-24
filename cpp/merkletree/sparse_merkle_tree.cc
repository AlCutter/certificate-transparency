#include "cpp/merkletree/sparse_merkle_tree.h"

#include <stddef.h>
#include <algorithm>
#include <vector>

#include "merkletree/merkle_tree_math.h"
#include "util/util.h"

using std::copy;
using std::fill;
using std::iterator;
using std::make_pair;
using std::ostream;
using std::ostringstream;
using std::reverse;
using std::string;
using std::unique_ptr;
using std::unordered_map;
using std::vector;


const vector<string>* GetNullHashes(const TreeHasher& hasher) {
  static unique_ptr<const vector<string>> null_hashes;
  if (!null_hashes) {
    vector<string> r{hasher.HashLeaf("")};
    for (int i(1); i < hasher.DigestSize() * 8; ++i) {
      r.emplace_back(hasher.HashChildren(r.back(), r.back()));
    }
    reverse(r.begin(), r.end());
    null_hashes.reset(new vector<string>(std::move(r)));
  }
  return null_hashes.get();
}


SparseMerkleTree::Path::Path() {
  fill(b_.begin(), b_.end(), 0);
}


SparseMerkleTree::Path::Path(uint64_t lsb) : Path() {
  for (int i(0); i < sizeof(lsb); ++i) {
    b_[b_.size() - 1 - i] |= lsb & 0xff;
    lsb >>= 8;
  }
}


SparseMerkleTree::Path::Path(const SparseMerkleTree::Path& other) : Path() {
  copy(other.b_.begin(), other.b_.end(), b_.begin());
}


size_t SparseMerkleTree::Path::size() const {
  return b_.size();
}


uint8_t& SparseMerkleTree::Path::operator[](size_t pos) {
  CHECK_LT(pos, b_.size());
  return b_[pos];
}


const uint8_t& SparseMerkleTree::Path::operator[](size_t pos) const {
  CHECK_LT(pos, b_.size());
  return b_[pos];
}


bool SparseMerkleTree::Path::operator==(
    const SparseMerkleTree::Path& rhs) const {
  return b_ == rhs.b_;
}


size_t SparseMerkleTree::Path::hash() const {
  size_t ret(0);
  for (int i(sizeof(size_t)); i > 0; --i) {
    ret <<= 8;
    ret |= b_[b_.size() - 1 - i];
  }
  return std::hash<uint64_t>()(ret);
}


SparseMerkleTree::SparseMerkleTree(SerialHasher* hasher)
    : serial_hasher_(CHECK_NOTNULL(hasher)->Create()),
      treehasher_(hasher),
      null_hashes_(GetNullHashes(treehasher_)) {
}


void SparseMerkleTree::EnsureHaveLevel(size_t level) {
  if (tree_.size() < (level + 1)) {
    tree_.resize(level + 1);
  }
}


void SparseMerkleTree::SetLeaf(const Path& path, const string& data) {
  CHECK_EQ(treehasher_.DigestSize(), path.size());
  // Mark the tree dirty:
  root_hash_.clear();
  string leaf_hash(treehasher_.HashLeaf(data));

  IndexType node_index;
  for (int depth(0); depth < kDigestSizeBits; ++depth) {
    node_index.set_nth_msb(depth, path.nth_msb(depth));
    EnsureHaveLevel(depth);
    auto it(tree_[depth].find(node_index));
    if (it == tree_[depth].end()) {
      CHECK(tree_[depth]
                .emplace(make_pair(node_index, TreeNode(path, leaf_hash)))
                .second);
      return;
    } else if (it->second.type_ == TreeNode::INTERNAL) {
      // Mark the internal node hash dirty
      it->second.hash_.clear();
    } else if (*it->second.path_ == path) {
      // replacement
      CHECK_EQ(TreeNode::LEAF, it->second.type_);
      it->second.hash_ = std::move(leaf_hash);
      return;
    } else {
      // restructure: push the existing node down a level and replace this one
      // with an INTERNAL node
      CHECK_LT(depth, kDigestSizeBits - 1) << "Oops, hit the end with path: "
                                           << node_index;
      EnsureHaveLevel(depth + 1);
      node_index.set_nth_msb(depth + 1, it->second.path_->nth_msb(depth + 1));
      CHECK(tree_[depth + 1]
                .emplace(make_pair(node_index, std::move(it->second)))
                .second);
      it->second.type_ = TreeNode::INTERNAL;
      it->second.hash_.clear();
    }
  }
  LOG(FATAL) << "Failed to set " << path << " to " << data;
}


void SparseMerkleTree::DumpTree(ostream* os, size_t depth,
                                IndexType index) const {
  if (tree_.size() <= depth) {
    return;
  }
  const string indent((depth + 1) * 2, '-');
  for (int side(0); side < 2; ++side) {
    index.set_nth_msb(depth + 1, side == 1);
    auto child(tree_[depth].find(index));
    if (child != tree_[depth].end()) {
      *os << indent << side << ": " << child->second.DebugString() << "\n";
      DumpTree(os, depth + 1, index);
    }
  }
}


string SparseMerkleTree::Dump() const {
  ostringstream ret;
  ret << "\nTree [Root: " << util::ToBase64(root_hash_) << "]:\n";
  if (!tree_.empty()) {
    DumpTree(&ret, 0, 0);
  }
  return ret.str();
}


string SparseMerkleTree::CalculateSubtreeHash(size_t depth, IndexType index) {
  if (tree_.size() <= depth) {
    return null_hashes_->at(depth);
  }

  auto it(tree_[depth].find(index));
  if (it != tree_[depth].end()) {
    switch (it->second.type_) {
      case TreeNode::INTERNAL: {
        if (!it->second.hash_.empty()) {
          return it->second.hash_;
        }
        IndexType left_child_index(index);
        IndexType right_child_index(index);
        right_child_index.set_nth_msb(depth + 1, true);
        const string left(CalculateSubtreeHash(depth + 1, left_child_index));
        const string right(CalculateSubtreeHash(depth + 1, right_child_index));
        it->second.hash_.assign(treehasher_.HashChildren(left, right));
        return it->second.hash_;
      }

      case TreeNode::LEAF: {
        string ret(it->second.hash_);
        for (int i(kDigestSizeBits - 1); i > depth; --i) {
          if (it->second.path_->nth_msb(i) == 0) {
            ret = treehasher_.HashChildren(ret, null_hashes_->at(i));
          } else {
            ret = treehasher_.HashChildren(null_hashes_->at(i), ret);
          }
        }
        // TODO(alcutter): maybe cache this?
        return ret;
      }
    }
    LOG(FATAL) << "Unknown node type " << it->second.type_ << " !";
  }

  return null_hashes_->at(depth);
}


string SparseMerkleTree::CurrentRoot() {
  if (root_hash_.empty()) {
    root_hash_ = treehasher_.HashChildren(CalculateSubtreeHash(0, 0),
                                          CalculateSubtreeHash(0, 1));
  }
  return root_hash_;
}


std::vector<string> SparseMerkleTree::InclusionProof(const Path& path) {
  // TODO(alcutter): implement
  LOG(FATAL) << "Not implemented.";
}


string SparseMerkleTree::TreeNode::DebugString() const {
  ostringstream os;
  os << "[TreeNode type: ";
  switch (type_) {
    case INTERNAL:
      os << "I";
      break;
    case LEAF:
      os << "L";
      break;
  }

  os << " hash: ";
  if (!hash_.empty()) {
    os << util::ToBase64(hash_);
  } else {
    os << "(unset)";
  }

  if (path_) {
    os << " path: ";
    os << *path_;
  }
  os << "]";
  return os.str();
}


ostream& operator<<(ostream& out, const SparseMerkleTree::Path& path) {
  for (size_t i(0); i < path.b_.size(); ++i) {
    uint8_t t(path.b_[i]);
    for (size_t b(8); b > 0; --b) {
      out << (t & 0x80 ? '1' : '0');
      t <<= 1;
    }
  }
  return out;
}