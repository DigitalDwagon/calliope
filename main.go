package main

import (
	"fmt"
	"github.com/CorentinB/warc"
	"github.com/go-rod/rod"
	"log"
	"path"
)

var version = "0.0.1-ALPHA"

func main() {
	println("Starting")
	var rotatorSettings = warc.NewRotatorSettings()

	rotatorSettings.OutputDirectory = path.Join(".", "warcs")
	rotatorSettings.Compression = "ZSTD"
	rotatorSettings.Prefix = "calliope"

	rotatorSettings.WarcinfoContent.Set("software", fmt.Sprintf("calliope/%s", version))
	rotatorSettings.WARCWriterPoolSize = 1
	rotatorSettings.WarcSize = float64(15360)
	// TODO: Should operator be "ArchiveTeam" eventually?
	rotatorSettings.WarcinfoContent.Set("operator", "DigitalDragon <warc@digitaldragon.dev>")

	dedupeOptions := warc.DedupeOptions{LocalDedupe: true, SizeThreshold: 30}

	HTTPClientSettings := warc.HTTPClientSettings{
		RotatorSettings:     rotatorSettings,
		DedupeOptions:       dedupeOptions,
		DecompressBody:      true,
		SkipHTTPStatusCodes: []int{429},
		VerifyCerts:         true,
		TempDir:             path.Join(".", "tmp"),
		FullOnDisk:          false,
		RandomLocalIP:       false,
		DisableIPv4:         false,
		DisableIPv6:         false,
		IPv6AnyIP:           false,
	}

	var warcClient, err = warc.NewWARCWritingHTTPClient(HTTPClientSettings)
	if err != nil {
		log.Fatal("Failed to start warc client!", err)
	}
	httpClient := &warcClient.Client

	println("Made WARC client, making Chrome instance")

	browser := rod.New().MustConnect()
	defer browser.MustClose()

	router := browser.HijackRequests()
	defer router.MustStop()

	router.MustAdd("*", func(ctx *rod.Hijack) {
		// LoadResponse runs the default request to the destination of the request.
		// Not calling this will require you to mock the entire response.
		// This can be done with the SetXxx (Status, Header, Body) functions on the
		// ctx.Response struct.
		_ = ctx.LoadResponse(httpClient, true)
		println(ctx.Response.Payload().ResponseCode, " = ", ctx.Request.URL().String())
	})

	go router.Run()
	browser.MustPage("https://wiki.archiveteam.org").MustWaitLoad()
	println("done!")

	warcClient.CloseIdleConnections()
	err = warcClient.Close()
	if err != nil {
		println(err.Error())
	}
}
