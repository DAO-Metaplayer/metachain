package main

import (
	_ "embed"

	"github.com/DAO-Metaplayer/metachain/command/root"
	"github.com/DAO-Metaplayer/metachain/licenses"
)

var (
	//go:embed LICENSE
	license string
)

func main() {
	licenses.SetLicense(license)

	root.NewRootCommand().Execute()
}
