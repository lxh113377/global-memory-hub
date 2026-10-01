package main

import (
	"embed"
	"io/fs"
)

// distEmbed 前端静态资源, 由 build.ps1 从 ../web/dist 同步而来 (embed 目录必须在 .go 文件所在树内)。
//
//go:embed all:dist
var distEmbed embed.FS

func distSub() (fs.FS, error) {
	return fs.Sub(distEmbed, "dist")
}
