// Package idgen 提供 128 位元全域唯一且可排序的 ID 產生器實作
// 
// 結構 (Big‑Endian)：
//  ┌────────┬────────────────────────┬──────────┬──────────┬──────────┐
//  │ 16 bits│        64 bits         │ 16 bits  │ 16 bits  │ 16 bits  │
//  │ Epoch  │  Timestamp(ms)         │ RegionID │ NodeID   │ Sequence │
//  └────────┴────────────────────────┴──────────┴──────────┴──────────┘
// 依序組成 128 位元 ID，具有以下特性：
//  * 全域唯一：Region+Node+Sequence 消除碰撞
//  * 時間有序：高位時間戳確保單調遞增（同 big‑endian 字典序）
//  * 去中心化：每節點本地產生，無需集中協調
//  * 時鐘回撥保護：發現時鐘回退時會等待或提升 Epoch，確保 ID 整體值仍遞增
//  * 可解析：可快速解碼出時間、區域、節點等資訊
package idgen

import (
    "encoding/base64"
    "encoding/binary"
    "encoding/hex"
    "errors"
    "fmt"
    "sync"
    "time"
)

// ------------- 常量設定 ------------- //

// CustomEpoch 定義時間戳起算點 (毫秒)，選用近期固定時間以縮短 timestamp 數值範圍
var CustomEpoch = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

const (
    epochBits     = 16
    timestampBits = 64
    regionBits    = 16
    nodeBits      = 16
    seqBits       = 16

    maxEpoch   = (1 << epochBits) - 1
    maxRegion  = (1 << regionBits) - 1
    maxNode    = (1 << nodeBits) - 1
    maxSequence = (1 << seqBits) - 1
)

// ------------- ID 型別定義 ------------- //

// ID 以 16 byte 陣列表現
// Big‑Endian 可確保在位元組序列上也與生成時間近似遞增
// 使用 string/[]byte 表示時，直接比較即可符合時間先後順序
type ID [16]byte

// Bytes 直接回傳 16-byte 陣列的切片 (不可修改)
func (id ID) Bytes() []byte {
    return id[:]
}

// Hex 回傳十六進位字串表示 (32 字元)
func (id ID) Hex() string {
    return hex.EncodeToString(id[:])
}

// Base64URL 回傳 Base64 URL‑safe 字串，長度 22
func (id ID) Base64URL() string {
    return base64.RawURLEncoding.EncodeToString(id[:])
}

// String 預設用 Hex 表示 (Implement fmt.Stringer)
func (id ID) String() string { return id.Hex() }

// Parse 解析 16‑byte 或 hex/base64 字串為 ID
func Parse(s string) (ID, error) {
    var id ID

    switch len(s) {
    case 16: // 原始 bytes (UTF‑8 會破壞，僅限程式內部)
        copy(id[:], []byte(s))
        return id, nil
    case 22: // base64 URL‑safe (22 bytes 可還原 16 bytes)
        b, err := base64.RawURLEncoding.DecodeString(s)
        if err != nil {
            return id, err
        }
        if len(b) != 16 {
            return id, errors.New("invalid base64 length")
        }
        copy(id[:], b)
        return id, nil
    case 32: // hex 編碼
        b, err := hex.DecodeString(s)
        if err != nil {
            return id, err
        }
        copy(id[:], b)
        return id, nil
    default:
        return id, fmt.Errorf("unsupported id string length %d", len(s))
    }
}

// Decode 欄位
func (id ID) Decode() (epoch uint16, tsMillis uint64, regionID, nodeID, seq uint16) {
    epoch = binary.BigEndian.Uint16(id[0:2])
    tsMillis = binary.BigEndian.Uint64(id[2:10])
    regionID = binary.BigEndian.Uint16(id[10:12])
    nodeID = binary.BigEndian.Uint16(id[12:14])
    seq = binary.BigEndian.Uint16(id[14:16])
    return
}

// ------------- 產生器實作 ------------- //

type Generator struct {
    mu         sync.Mutex // 保護下列欄位的並發存取
    regionID   uint16
    nodeID     uint16
    epoch      uint16
    lastMillis uint64
    sequence   uint16
}

// NewGenerator 建立新的 Generator
// 需指定唯一的 regionID 與 nodeID，範圍 0‑65535
func NewGenerator(regionID, nodeID uint16) (*Generator, error) {
    if regionID > maxRegion {
        return nil, fmt.Errorf("region id %d 超出範圍 0‑%d", regionID, maxRegion)
    }
    if nodeID > maxNode {
        return nil, fmt.Errorf("node id %d 超出範圍 0‑%d", nodeID, maxNode)
    }
    return &Generator{regionID: regionID, nodeID: nodeID}, nil
}

// Next 產生下一個唯一且有序的 ID (thread‑safe)
func (g *Generator) Next() (ID, error) {
    g.mu.Lock()
    defer g.mu.Unlock()

    now := uint64(time.Now().UnixMilli() - CustomEpoch)

    // 時鐘回撥處理
    if now < g.lastMillis {
        // 若回撥幅度小 (< 5ms)，等待時間追上；否則升級 epoch
        drift := g.lastMillis - now
        if drift <= 5 {
            time.Sleep(time.Duration(drift) * time.Millisecond)
            now = uint64(time.Now().UnixMilli() - CustomEpoch)
            if now < g.lastMillis { // 還是無法追上，保險做 epoch++
                g.epoch = (g.epoch + 1) & maxEpoch
            }
        } else {
            g.epoch = (g.epoch + 1) & maxEpoch
            now = g.lastMillis // 確保值不減小
        }
    }

    if now == g.lastMillis {
        g.sequence++
        if g.sequence > maxSequence {
            // 序列號溢出：等待下一毫秒
            for now <= g.lastMillis {
                time.Sleep(time.Millisecond)
                now = uint64(time.Now().UnixMilli() - CustomEpoch)
            }
            g.sequence = 0
        }
    } else {
        g.sequence = 0
    }

    g.lastMillis = now

    // 組裝 ID (Big‑Endian)：
    var id ID
    binary.BigEndian.PutUint16(id[0:2], g.epoch)
    binary.BigEndian.PutUint64(id[2:10], now)
    binary.BigEndian.PutUint16(id[10:12], g.regionID)
    binary.BigEndian.PutUint16(id[12:14], g.nodeID)
    binary.BigEndian.PutUint16(id[14:16], g.sequence)

    return id, nil
}

// ------------- 使用範例 ------------- //

/*
func main() {
    gen, err := NewGenerator(1, 42)
    if err != nil {
        panic(err)
    }

    id, _ := gen.Next()
    fmt.Println("Hex:", id.Hex())
    fmt.Println("Base64URL:", id.Base64URL())

    ep, ts, region, node, seq := id.Decode()
    fmt.Printf("epoch=%d ts=%d region=%d node=%d seq=%d\n", ep, ts, region, node, seq)
}
*/
