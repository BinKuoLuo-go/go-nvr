/**
@Time : 2026/03/10 09:28
@Author: FangYao( 方少、)
@Description: 帧池
@Email: fy20030315@163.com
*/

package ffmpeg

import (
	"image"
	"sync"
)

var rgbaPool = sync.Pool{
	New: func() any {
		return &image.RGBA{}
	},
}

var rgbaClonePool = sync.Pool{
	New: func() any {
		return &image.RGBA{}
	},
}

func GetRGBA(rect image.Rectangle) *image.RGBA {

	img := rgbaPool.Get().(*image.RGBA)

	size := rect.Dx() * rect.Dy() * 4

	if cap(img.Pix) < size {
		img.Pix = make([]byte, size)
	}

	img.Pix = img.Pix[:size]
	img.Stride = rect.Dx() * 4
	img.Rect = rect

	return img
}

func PutRGBA(img *image.RGBA) {
	if img != nil {
		rgbaPool.Put(img)
	}
}

func GetRGBAClone(rect image.Rectangle) *image.RGBA {

	img := rgbaClonePool.Get().(*image.RGBA)

	size := rect.Dx() * rect.Dy() * 4

	if cap(img.Pix) < size {
		img.Pix = make([]byte, size)
	}

	img.Pix = img.Pix[:size]
	img.Stride = rect.Dx() * 4
	img.Rect = rect

	return img
}

func PutRGBAClone(img *image.RGBA) {
	if img != nil {
		rgbaClonePool.Put(img)
	}
}
