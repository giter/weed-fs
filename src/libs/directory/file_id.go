package directory

import (
  "encoding/hex"
  "log"
  "libs/storage"
  "strconv"
  "strings"
  "libs/util"
)

type FileId struct {
  VolumeId uint32
  Key      uint64
  Hashcode uint32
}

func NewFileId(VolumeId uint32, Key uint64, Hashcode uint32) *FileId {

  return &FileId{VolumeId: VolumeId, Key: Key, Hashcode: Hashcode}
}

func ParseFileId(fid string) *FileId {

  a := strings.Split(fid, ",")

  if len(a) != 2 {
    log.Println("Invalid fid", fid, ", split length", len(a))
    return nil
  }

  vid_string, key_hash_string := a[0], a[1]
  vid, _ := strconv.ParseUint(vid_string,10,64)
  key, hash := storage.ParseKeyHash(key_hash_string)

  return &FileId{VolumeId: uint32(vid), Key: key, Hashcode: hash}
}

func (n *FileId) String() string {

  bytes := make([]byte, 12)
  util.Uint64toBytes(bytes[0:8], n.Key)
  util.Uint32toBytes(bytes[8:12], n.Hashcode)
  nonzero_index := 0

  for ; bytes[nonzero_index] == 0; nonzero_index++ {
  }

  return strconv.FormatUint(uint64(n.VolumeId),10) + "," + hex.EncodeToString(bytes[nonzero_index:])
}
