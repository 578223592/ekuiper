package main

import (
	_ "encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/lf-edge/ekuiper/v2/pkg/cast"
)

type MqttPayLoadStrSlice struct {
	Data []string `json:"data"`
}

type MqttPayLoadByteSlice struct {
	Data []byte `json:"data"`
}

type MqttPayLoadFloat32Slice struct {
	Data []float32 `json:"data"`
}

// MarshalJSON 自定义 JSON 序列化
func TestPic(t *testing.T) {
	const TOPIC = "onnxPubImg"

	images := []string{
		
		"img.png",
		// 其他你需要的图像
	}
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	for _, image := range images {
		fmt.Println("Publishing " + image)
		inputImage, err := NewProcessedImage(image, false)

		if err != nil {
			fmt.Println(err)
			continue
		}
		// payload, err := os.ReadFile(image)
		payloadF32 := inputImage.GetNetworkInput()

		data := make([]any, len(payloadF32))
		for i := 0; i < len(data); i++ {
			data[i] = payloadF32[i]
		}
		payloadUnMarshal := MqttPayLoadFloat32Slice{
			Data: payloadF32,
		}
		payload, err := json.Marshal(payloadUnMarshal)
		if err != nil {
			fmt.Println(err)
			continue
		} else {
			fmt.Println(string(payload))
		}
		if token := client.Publish(TOPIC, 2, true, payload); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		} else {
			fmt.Println("Published " + image)
		}
		time.Sleep(1 * time.Second)
	}
	client.Disconnect(0)
}

func TestBytes(t *testing.T) {
	const TOPIC = "onnxPubImg"

	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	payloadUnMarshal := MqttPayLoadStrSlice{
		Data: []string{"hello world"},
	}
	payload, err := json.Marshal(payloadUnMarshal)
	if err != nil {
		fmt.Println(err)

	} else {
		fmt.Println(string(payload))
	}
	if token := client.Publish(TOPIC, 2, false, payload); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	client.Disconnect(0)
}

func TestBytesOri(t *testing.T) {
	const TOPIC = "onnxPubImg"

	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	if token := client.Publish(TOPIC, 2, false, []byte{
		4, 2, 3,
	}); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	client.Disconnect(0)
}

func TestStringOri(t *testing.T) {
	const TOPIC = "onnxPubImg"

	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	paylod := "clienttest"

	if token := client.Publish(TOPIC, 2, false, paylod); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
	}

	client.Disconnect(0)
}

func TestBytes2Float32(t *testing.T) {
	value, err := cast.ToFloat32Slice([]byte(`[1.1, 2.2, 3.3]`), cast.CONVERT_SAMEKIND)
	if err != nil { 
		 t.Errorf("invalid with err %v", err)
	}
	fmt.Println(value)

	// 创建一个空的 []float32 类型的切片
    var float32Slice []float32

    // 将 JSON 数据解码到 []float32
    err = json.Unmarshal([]byte(`[1.1, 2.2, 3.3]`), &float32Slice)
	fmt.Println(float32Slice)
}

// func TestSlicePrint(t *testing.T) {
// 	testSlice := make([]float32, 5)
// 	fmt.Println(testSlice)
// }

// func TestSlicePrint(t *testing.T) {
// 	arg := any()

// 	switch v := arg.(type) {
// 		case []any: // only supports one dimensional arg. Even dim 0 must be an array of 1 element
// 			arg = v
// 		case string: //string is also array of bytea
// 			arg = append(arg, v)
// 		default:
// 			fmt.Println(args[i])
// 			return fmt.Errorf("onnx function parameter %d must be a bytea or array of bytea, but got %[1]T(%[1]v)", i, args[i]), false
// 		}
// }

// Takes a path to an image file, loads the image, and returns a ProcessedImage
// struct which can be used to obtain the neural network input.
func NewProcessedImage(path string, invertBrightness bool) (*ProcessedImage,
	error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, fmt.Errorf("Error opening %s: %w", path, e)
	}
	defer f.Close()
	originalPic, _, e := image.Decode(f)
	if e != nil {
		return nil, fmt.Errorf("Error decoding image %s: %w", path, e)
	}
	bounds := originalPic.Bounds().Canon()
	if (bounds.Min.X != 0) || (bounds.Min.Y != 0) {
		// Should never happen with the standard library.
		return nil, fmt.Errorf("Bounding rect of %s doesn't start at 0, 0",
			path)
	}
	return &ProcessedImage{
		dx:     float32(bounds.Dx()) / 28.0,
		dy:     float32(bounds.Dy()) / 28.0,
		pic:    originalPic,
		Invert: invertBrightness,
	}, nil
}

// Used to satisfy the image interface as well as to help with formatting and
// resizing an input image into the format expected as a network input.
type ProcessedImage struct {
	// The number of "pixels" in the input image corresponding to a single
	// pixel in the 28x28 output image.
	dx, dy float32

	// The input image being transformed
	pic image.Image

	// If true, the grayscale values in the postprocessed image will be
	// inverted, so that dark colors in the original become light, and vice
	// versa. Recall that the network expects black backgrounds, so this should
	// be set to true for images with light backgrounds.
	Invert bool
}

// Returns a slice of data that can be used as the input to the onnx network.
func (p *ProcessedImage) GetNetworkInput() []float32 {
	toReturn := make([]float32, 0, 28*28)
	for row := 0; row < 28; row++ {
		for col := 0; col < 28; col++ {
			c := float32(p.At(col, row).(grayscaleFloat))
			toReturn = append(toReturn, c)
		}
	}
	return toReturn
}

// Returns an average grayscale value using the pixels in the input image.
func (p *ProcessedImage) At(x, y int) color.Color {
	if (x < 0) || (x >= 28) || (y < 0) || (y >= 28) {
		return color.Black
	}

	// Compute the window of pixels in the input image we'll be averaging.
	startX := int(float32(x) * p.dx)
	endX := int(float32(x+1) * p.dx)
	if endX == startX {
		endX = startX + 1
	}
	startY := int(float32(y) * p.dy)
	endY := int(float32(y+1) * p.dy)
	if endY == startY {
		endY = startY + 1
	}

	// Compute the average brightness over the window of pixels
	var sum float32
	var nPix int
	for row := startY; row < endY; row++ {
		for col := startX; col < endX; col++ {
			c := p.pic.At(col, row)
			grayValue := color.Gray16Model.Convert(c).(color.Gray16).Y
			sum += float32(grayValue) / 0xffff
			nPix++
		}
	}

	brightness := grayscaleFloat(sum / float32(nPix))
	if p.Invert {
		brightness = 1.0 - brightness
	}
	return brightness
}

// Implements the color interface
type grayscaleFloat float32

func (f grayscaleFloat) RGBA() (r, g, b, a uint32) {
	a = 0xffff
	v := uint32(f * 0xffff)
	if v > 0xffff {
		v = 0xffff
	}
	r = v
	g = v
	b = v
	return
}


// Returns the brightness of the image
func (p *ProcessedImage) Brightness() float32 {
	var sum float32
	for y := 0; y < 28; y++ {
		for x := 0; x < 28; x++ {
			c := p.At(x, y).(grayscaleFloat)
			sum += float32(c)
		}
	}

	return sum / (28 * 28)


}

