package main

import (
	"crypto/rand"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type captchaEntry struct {
	equation string
	answer   int
	created  time.Time
}

var captchaStore = struct {
	sync.RWMutex
	data map[string]captchaEntry
}{data: make(map[string]captchaEntry)}

func main() {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		equation, answer := generateEquation()
		captchaID := uuid.New().String()

		captchaStore.Lock()
		captchaStore.data[captchaID] = captchaEntry{
			equation: equation,
			answer:   answer,
			created:  time.Now(),
		}
		captchaStore.Unlock()

		c.HTML(http.StatusOK, "index.html", gin.H{
			"captchaID": captchaID,
		})
	})

	r.GET("/captcha/new", func(c *gin.Context) {
		equation, answer := generateEquation()
		captchaID := uuid.New().String()

		captchaStore.Lock()
		captchaStore.data[captchaID] = captchaEntry{
			equation: equation,
			answer:   answer,
			created:  time.Now(),
		}
		captchaStore.Unlock()

		c.JSON(http.StatusOK, gin.H{
			"captchaID": captchaID,
			"imageUrl":  "/captcha/image/" + captchaID,
		})
	})

	r.GET("/captcha/image/:id", func(c *gin.Context) {
		captchaID := c.Param("id")

		captchaStore.RLock()
		entry, exists := captchaStore.data[captchaID]
		captchaStore.RUnlock()

		if !exists {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		img := generateImage(entry.equation)
		c.Header("Content-Type", "image/png")
		png.Encode(c.Writer, img)
	})

	r.POST("/validate", func(c *gin.Context) {
		captchaID := c.PostForm("captchaID")
		userAnswer := c.PostForm("answer")

		captchaStore.Lock()
		entry, exists := captchaStore.data[captchaID]
		if exists {
			delete(captchaStore.data, captchaID)
		}
		captchaStore.Unlock()

		if !exists {
			c.HTML(http.StatusBadRequest, "index.html", gin.H{
				"error": "CAPTCHA expired or invalid",
			})
			return
		}

		userAnswerInt, err := strconv.Atoi(userAnswer)
		if err != nil || userAnswerInt != entry.answer {
			c.HTML(http.StatusBadRequest, "index.html", gin.H{
				"error": "Incorrect answer, please try again",
			})
			return
		}

		c.HTML(http.StatusOK, "success.html", nil)
	})

	r.Run(":8080")
}

func generateEquation() (string, int) {
	ops := []string{"+", "-", "*"}
	opIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(ops))))
	op := ops[opIndex.Int64()]

	var a, b int
	switch op {
	case "+":
		a = getRandomNumber(1, 10)
		b = getRandomNumber(1, 10)
	case "-":
		a = getRandomNumber(1, 20)
		b = getRandomNumber(1, a)
	case "*":
		a = getRandomNumber(1, 10)
		b = getRandomNumber(1, 10)
	}

	equation := fmt.Sprintf("%d %s %d = ?", a, op, b)
	answer := calculateAnswer(a, b, op)
	return equation, answer
}

func getRandomNumber(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func calculateAnswer(a, b int, op string) int {
	switch op {
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	default:
		return 0
	}
}

func generateImage(text string) *image.RGBA {
	width, height := 200, 80
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), image.White, image.Point{}, draw.Src)

	ttfFont, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic(err)
	}

	face, err := opentype.NewFace(ttfFont, &opentype.FaceOptions{
		Size:    32,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}

	textWidth := font.MeasureString(face, text)
	startX := (fixed.I(width) - textWidth) / 2

	d := font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{0, 0, 0, 255}),
		Face: face,
		Dot:  fixed.P(startX.Ceil(), 50),
	}
	d.DrawString(text)

	addDistortion(img, width, height)
	return img
}
func addDistortion(img *image.RGBA, width, height int) {
	for i := 0; i < 10; i++ {
		x1 := randInt(width)
		y1 := randInt(height)
		x2 := randInt(width)
		y2 := randInt(height)
		drawLine(img, x1, y1, x2, y2, color.Black)
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, color color.Color) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		img.Set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func randInt(max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max)))
	return int(n.Int64())
}
