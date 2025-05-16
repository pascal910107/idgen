# 128‑bit Distributed, Time‑Ordered ID Generator

> **Languages / 語言**：English | 中文

---

## Overview

This repository provides an implementation of a **128‑bit globally unique, time‑ordered identifier (ID)** generator written in Go.
It blends the advantages of Snowflake, KSUID, and UUID‑v7 while adding clock‑rollback protection and flexible bit allocation.

---

## Features / 特性

| Feature                   | Description                                                             | 特性說明                       |
| ------------------------- | ----------------------------------------------------------------------- | -------------------------- |
| **Globally unique**       | Region + Node + Sequence guarantee collision‑free IDs                   | 區域 + 節點 + 序列確保不碰撞          |
| **Time‑ordered**          | 64‑bit millisecond timestamp at the high bits → numeric & lexical order | 高位 64 位毫秒級時間戳 → 數值與字典序單調遞增 |
| **Clock‑rollback safety** | Wait‑or‑bump‑epoch strategy prevents duplicates                         | 時鐘回撥時等待或提升 Epoch 保序且不重覆    |
| **No central server**     | IDs generated locally; horizontal scalability                           | 去中心化本地產生；水平擴充無上限           |
| **Human‑decodable**       | Quick decode to timestamp / region / node                               | 可解析生成時間、區域與節點資訊            |
| **MIT License**           | Free use in commercial & OSS projects                                   | MIT 授權，商用／開源皆可使用           |

---

## Getting Started / 快速開始

```bash
# Requires Go 1.22+
$ go get github.com/pascal910107/idgen
```

```go
package main

import (
    "fmt"
    "github.com/pascal910107/idgen"
)

func main() {
    g, _ := idgen.NewGenerator(1, 42) // region=1, node=42
    id, _ := g.Next()
    fmt.Println("ID (Hex):", id)
    fmt.Println("ID (Base64):", id.Base64URL())

    // Decode for debugging
    ep, ts, r, n, seq := id.Decode()
    fmt.Printf("epoch=%d ts(ms)=%d region=%d node=%d seq=%d\n", ep, ts, r, n, seq)
}
```

### Build & Run / 編譯與執行

```bash
$ go run ./examples/main.go
```

### Configuration / 參數設定

| Field     | Bits | Range         | 說明            |
| --------- | ---- | ------------- | ------------- |
| Epoch     | 16   | 0–65 535      | 時鐘回撥／世代號      |
| Timestamp | 64   | ≥ 584 億年 (ms) | 自定 Epoch 起毫秒差 |
| Region ID | 16   | 0–65 535      | 資料中心或雲區域代號    |
| Node ID   | 16   | 0–65 535      | 同區域內節點唯一 ID   |
| Sequence  | 16   | 0–65 535      | 同毫秒內序列號       |

See `id_generator.go` for full documentation.

---

## Roadmap / 待辦

* [ ] CLI tool for batch ID generation
* [ ] Drivers for Java / Rust / Python
* [ ] Docker image & Helm chart

---

## License / 授權

This project is licensed under the **MIT License**.
本專案採用 **MIT 授權**，可自由商業或非商業使用。
