package ising

import (
	"bytes"
	"encoding/binary"

	. "github.com/nknorg/nkn/common"
	"github.com/nknorg/nkn/crypto"
	"github.com/nknorg/nkn/util/log"
)

func publickKeyToNodeID(pubKey *crypto.PubKey) uint64 {
	var id uint64
	key, err := pubKey.EncodePoint(true)
	if err != nil {
		log.Error(err)
	}
	err = binary.Read(bytes.NewBuffer(key[:8]), binary.LittleEndian, &id)
	if err != nil {
		log.Error(err)
	}

	return id
}

// HeightHashToString uses block height and block hash to generate a uniq string
func HeightHashToString(height uint32, blockHash Uint256) string {
	buff := bytes.NewBuffer(nil)
	binary.Write(buff, binary.LittleEndian, height)
	buff.Write(blockHash.ToArray())

	return buff.String()
}

// StringToHeightHash recovers block height and block hash from string which generated by HeightHashToString
func StringToHeightHash(str string) (uint32, Uint256) {
	var height uint32
	var hash Uint256
	buff := bytes.NewReader([]byte(str))
	binary.Read(buff, binary.LittleEndian, &height)
	hash.Deserialize(buff)

	return height, hash
}