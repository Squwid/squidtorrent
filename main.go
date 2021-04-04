package main

import (
	"os"
	"time"

	"github.com/Squwid/squidtorrent/torrentfile"
	"github.com/go-echarts/statsview"
	"github.com/go-echarts/statsview/viewer"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	viewer.SetConfiguration(viewer.WithAddr("192.168.1.191:18066"))

	mgr := statsview.New()
	go mgr.Start()
	logger.Infof("Metrics: http://192.168.1.191:18066/debug/statsview")
	time.Sleep(5 * time.Second)

	inPath := os.Args[1]
	outPath := os.Args[2]

	tf, err := torrentfile.Open(inPath)
	if err != nil {
		logger.WithError(err).Errorf("Error opening torrent file")
	}

	err = tf.DownloadToFile(outPath)
	if err != nil {
		logger.WithError(err).Errorf("Error downloading torrent file")
	}
}
