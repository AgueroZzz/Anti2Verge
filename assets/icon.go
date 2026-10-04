package assets

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"math"
)

// GenerateAntigravityImage 绘制一个高清具有科技质感的 Antigravity 蓝紫色图标
func GenerateAntigravityImage(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	center := float64(size) / 2.0
	radius := float64(size)/2.0 - math.Max(1.0, float64(size)*0.04)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - center
			dy := float64(y) - center
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= radius {
				// 外边缘平滑抗锯齿
				alpha := 1.0
				if dist > radius-1.0 {
					alpha = math.Max(0.0, radius-dist)
				}

				// 球体光照模型 (左上方光源渐变)
				lightX := center - radius*0.35
				lightY := center - radius*0.35
				distLight := math.Sqrt((float64(x)-lightX)*(float64(x)-lightX) + (float64(y)-lightY)*(float64(y)-lightY))
				lightFactor := math.Max(0.0, 1.0-distLight/(radius*1.4))

				// Antigravity 标志性科技蓝紫色渐变 (RGB: 40,110,240 -> 120,40,210)
				r := uint8(math.Min(255, 45+145*lightFactor))
				g := uint8(math.Min(255, 115+115*lightFactor))
				b := uint8(math.Min(255, 235+20*lightFactor))

				// 中间绘制一个亮白色的重力/桥接双翼标志
				normX := (float64(x) - center) / (radius * 0.65)
				normY := (float64(y) - center) / (radius * 0.65)

				isSymbol := false
				// 判定是否在 V 形科技桥梁笔画内
				if normY >= -0.65 && normY <= 0.65 {
					expectedY := math.Abs(normX)*1.5 - 0.4
					if math.Abs(normY-expectedY) < 0.26 && math.Abs(normX) < 0.8 {
						isSymbol = true
					}
				}

				if isSymbol {
					img.Set(x, y, color.RGBA{
						R: 255,
						G: 255,
						B: 255,
						A: uint8(255 * alpha),
					})
				} else {
					img.Set(x, y, color.RGBA{
						R: r,
						G: g,
						B: b,
						A: uint8(255 * alpha),
					})
				}
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 0})
			}
		}
	}

	return img
}

// dibEntry 存储单个尺寸的 DIB 图像数据和参数
type dibEntry struct {
	width          int
	height         int
	imageBytesSize int
	data           []byte
}

func encodeSingleDIB(img *image.RGBA) dibEntry {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	maskRowBytes := ((width + 31) / 32) * 4
	maskSize := maskRowBytes * height
	xorSize := width * height * 4
	bihSize := 40
	imageBytesSize := bihSize + xorSize + maskSize

	var data bytes.Buffer

	// BITMAPINFOHEADER (40 字节)
	binary.Write(&data, binary.LittleEndian, uint32(40))
	binary.Write(&data, binary.LittleEndian, int32(width))
	binary.Write(&data, binary.LittleEndian, int32(height*2)) // biHeight = 2 * height (XOR + AND)
	binary.Write(&data, binary.LittleEndian, uint16(1))
	binary.Write(&data, binary.LittleEndian, uint16(32)) // 32bpp
	binary.Write(&data, binary.LittleEndian, uint32(0))  // BI_RGB
	binary.Write(&data, binary.LittleEndian, uint32(xorSize))
	binary.Write(&data, binary.LittleEndian, int32(0))
	binary.Write(&data, binary.LittleEndian, int32(0))
	binary.Write(&data, binary.LittleEndian, uint32(0))
	binary.Write(&data, binary.LittleEndian, uint32(0))

	// XOR 掩码 (BGRA, 自底向上)
	for y := height - 1; y >= 0; y-- {
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			data.WriteByte(c.B)
			data.WriteByte(c.G)
			data.WriteByte(c.R)
			data.WriteByte(c.A)
		}
	}

	// AND 掩码 (单色，自底向上)
	andMask := make([]byte, maskSize)
	for y := height - 1; y >= 0; y-- {
		rowOffset := (height - 1 - y) * maskRowBytes
		for x := 0; x < width; x++ {
			c := img.RGBAAt(x, y)
			if c.A < 16 {
				byteIdx := rowOffset + (x / 8)
				bitIdx := uint(7 - (x % 8))
				andMask[byteIdx] |= (1 << bitIdx)
			}
		}
	}
	data.Write(andMask)

	return dibEntry{
		width:          width,
		height:         height,
		imageBytesSize: imageBytesSize,
		data:           data.Bytes(),
	}
}

// EncodeMultiSizeICO 生成包含多种分辨率的高清 Windows ICO 格式
func EncodeMultiSizeICO(sizes []int) []byte {
	if len(sizes) == 0 {
		sizes = []int{16, 32, 48, 64}
	}

	entries := make([]dibEntry, 0, len(sizes))
	for _, sz := range sizes {
		img := GenerateAntigravityImage(sz)
		entries = append(entries, encodeSingleDIB(img))
	}

	var buf bytes.Buffer

	// ICONHEADER (6 字节)
	binary.Write(&buf, binary.LittleEndian, uint16(0))               // 保留
	binary.Write(&buf, binary.LittleEndian, uint16(1))               // 1 = ICO
	binary.Write(&buf, binary.LittleEndian, uint16(len(entries)))     // 图像数量

	// 计算首个图像数据的起始偏移量 (6 头部 + 16 * 图像数)
	currentOffset := uint32(6 + 16*len(entries))

	// 依次写入每个 ICONDIRENTRY (16 字节)
	for _, entry := range entries {
		w := byte(entry.width)
		if entry.width >= 256 {
			w = 0
		}
		h := byte(entry.height)
		if entry.height >= 256 {
			h = 0
		}
		buf.WriteByte(w)
		buf.WriteByte(h)
		buf.WriteByte(0)
		buf.WriteByte(0)
		binary.Write(&buf, binary.LittleEndian, uint16(1))
		binary.Write(&buf, binary.LittleEndian, uint16(32))
		binary.Write(&buf, binary.LittleEndian, uint32(entry.imageBytesSize))
		binary.Write(&buf, binary.LittleEndian, currentOffset)

		currentOffset += uint32(entry.imageBytesSize)
	}

	// 依次追加每个图像的 DIB 数据
	for _, entry := range entries {
		buf.Write(entry.data)
	}

	return buf.Bytes()
}

// GenerateDefaultIcon 生成多尺寸 (16, 32, 48, 64) 的标准 Windows ICO 数据
func GenerateDefaultIcon() []byte {
	return EncodeMultiSizeICO([]int{16, 32, 48, 64})
}
