package vimg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	vimgImageBuffer = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vimg_imagebuffer",
		Help: "ImageBuffer requests",
	},[]string{"action","type"})
)

var (
	vimgOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vimg_operations",
		Help: "VIPS Operations",
	},[]string{"type"})
)