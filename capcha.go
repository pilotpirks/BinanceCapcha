package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/stealth"
	"github.com/orisano/pixelmatch"
	"gocv.io/x/gocv"
)

const (
	userEmail    = "user@mail.com"
	userPassword = "userPassword"

	url = "https://www.binance.us/en/home"

	// buttons
	loginButton = "#__APP > div > div > div.css-1pthsua > div.css-6f0qf8 > header > div.css-4cffwv > a"

	elSlider = "body > div.geetest_panel.geetest_wind > div.geetest_panel_box.geetest_no_logo.geetest_panelshowslide > div.geetest_panel_next > div > div.geetest_wrap > div.geetest_slider.geetest_ready > div.geetest_slider_button"

	elRetrySlider = "#captcha > div.geetest_holder.geetest_wind.geetest_radar_error > div.geetest_btn > div.geetest_radar_btn > div.geetest_radar_tip > span.geetest_reset_tip_content"

	// all images, related to the captcha
	// "bg.png" "fullbg.png" "puzzle.png"
	bgImage = "body > div.geetest_panel.geetest_wind > div.geetest_panel_box.geetest_no_logo.geetest_panelshowslide > div.geetest_panel_next > div > div.geetest_wrap > div.geetest_widget > div > a > div.geetest_canvas_img.geetest_absolute > div > canvas.geetest_canvas_bg.geetest_absolute"

	fullbgImage = "body > div.geetest_panel.geetest_wind > div.geetest_panel_box.geetest_no_logo.geetest_panelshowslide > div.geetest_panel_next > div > div.geetest_wrap > div.geetest_widget > div > a > div.geetest_canvas_img.geetest_absolute > canvas"

	// puzzleImage = "body > div.geetest_fullpage_click.geetest_float.geetest_wind.geetest_slide3 > div.geetest_fullpage_click_wrap > div.geetest_fullpage_click_box > div > div.geetest_wrap > div.geetest_widget > div > a > div.geetest_canvas_img.geetest_absolute > div > canvas.geetest_canvas_slice.geetest_absolute"
)

var (
	browser *rod.Browser
	page    *rod.Page
)

func randInt(min, max int) int {
	return rand.Intn(max-min+1) + min
}

func openImage(imagePath string) (image.Image, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("err: %s", err)
		return nil, err
	}

	image, _, err := image.Decode(file)
	if err != nil {
		log.Printf("err: %s, %s", imagePath, err)
		return nil, err
	}

	return image, nil
}

func handleError(err error) {
	var evalErr *rod.ErrEval
	if errors.Is(err, context.DeadlineExceeded) { // timeout error
		log.Println("timeout err")
	} else if errors.As(err, &evalErr) { // eval error
		log.Println(evalErr.LineNumber)
	} else if err != nil {
		log.Println("can't handle", err)
	}
}

func getCaptchaImages() {

	page.MustNavigate(url).MustWaitLoad()
	page.MustElement(loginButton).MustClick()
	page.MustWaitLoad()

	page.MustElement("input[name=email]").MustInput(userEmail)
	page.MustElement("input[name=password]").MustInput(userPassword).MustPress(input.Enter)

	el, err := page.ElementR(".geetest_slider", "Slide to complete the puzzle") // "Slide to complete the puzzle"
	if err != nil {
		handleError(err)
		return
	}
	el.MustClick()

	canvaces := map[string]string{
		"bg":     bgImage,
		"fullbg": fullbgImage,
		// "puzzle": puzzleImage,
	}

	for key, val := range canvaces {

		el, err := page.Element(val)
		if err != nil {
			handleError(err)
			return
		}
		img, err := el.CanvasToImage("image/png", 100)
		if err != nil {
			handleError(err)
			return
		}

		newImg, _, err := image.Decode(bytes.NewReader(img))
		if err != nil {
			log.Println("image.Decode", err)
			continue
		}

		out, err := os.Create(key + ".png")
		if err != nil {
			log.Println("os.Create", err)
			continue
		}

		err = png.Encode(out, newImg)
		if err != nil {
			log.Println("png.Encode", err)
			continue
		}
	}
}

func findCenterImage() (image.Point, error) {
	var center image.Point

	diffImage := gocv.IMRead("diff.png", gocv.IMReadColor)
	if diffImage.Empty() {
		return center, fmt.Errorf("can't open diff image")
	}

	// фильтры
	thresholdedImage := gocv.NewMat()
	defer thresholdedImage.Close()

	erodedImage := gocv.NewMat()
	defer erodedImage.Close()

	dilatedImage := gocv.NewMat()
	defer dilatedImage.Close()

	grayImage := gocv.NewMat()
	defer grayImage.Close()

	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(4, 4))
	defer kernel.Close()

	gocv.Threshold(diffImage, &thresholdedImage, 127, 255, gocv.ThresholdBinary)
	gocv.Erode(thresholdedImage, &erodedImage, kernel)
	gocv.Dilate(erodedImage, &dilatedImage, kernel)

	gocv.CvtColor(dilatedImage, &grayImage, gocv.ColorBGRToGray)
	gocv.Threshold(grayImage, &grayImage, 150, 255, gocv.ThresholdBinaryInv)

	// fragment position: [(122,43) (119,46)...]
	contours := gocv.FindContours(grayImage, gocv.RetrievalExternal, gocv.ChainApproxSimple)
	contour := contours.At(0) // есть только одна фигура

	// fragment center pazzle: x,y (126,67)
	center = gocv.MinAreaRect(contour).Center

	return center, nil
}

func saveDiffImage() {
	bg, err1 := openImage("bg.png")
	fullbg, err2 := openImage("fullbg.png")
	if err1 != nil || err2 != nil {
		return
	}

	log.Println("bg.Bounds()", bg.Bounds())

	var diffImage image.Image

	opts := []pixelmatch.MatchOption{
		pixelmatch.Threshold(0.1),
		pixelmatch.IncludeAntiAlias,
		pixelmatch.WriteTo(&diffImage),
		// pixelmatch.Alpha(*alpha),
		// pixelmatch.AntiAliasedColor(color.RGBA(antiAliased)),
		// pixelmatch.DiffColor(color.RGBA(diffColor)),
	}

	_, err := pixelmatch.MatchPixel(bg, fullbg, opts...)
	if err != nil {
		log.Printf("err: %s", err)
		return
	}

	var w io.Writer
	dest := "diff.png"

	f, err := os.Create(dest)
	if err != nil {
		log.Printf("create destination image: %s", err)
		return
	}
	defer f.Close()
	w = f

	err = png.Encode(w, diffImage)
	if err != nil {
		log.Printf("Encode error: %s", err)
		return
	}
}

func slide(el *rod.Element, posX, posY float64) error {
	mouse := page.Mouse
	mouseStep := randInt(5, 8)
	mouseSleep := randInt(50, 150)

	err := el.Hover()
	if err != nil {
		handleError(err)
		return err
	}

	err = mouse.Down("left", 0)
	if err != nil {
		log.Println("mouse.Down", err)
		return err
	}
	time.Sleep(time.Millisecond * time.Duration(mouseSleep))

	err = mouse.Move(posX, posY, mouseStep)
	if err != nil {
		log.Println("mouse.move", err)
		return err
	}

	err = mouse.Up("left", 0)
	if err != nil {
		log.Println("mouse.Up", err)
		return err
	}

	return nil
}

func moveCapchaSlider(center image.Point) error {

	el, err := page.Element(elSlider)

	if err != nil {
		handleError(err)
		return err
	}

	box, err := el.Shape()
	if err != nil {
		log.Println("get rect", err)
		return err
	}
	rect := box.Box()

	elCenterX := rect.X + rect.Width/2
	elCenterY := rect.Y + rect.Height/3
	log.Println("box", rect)

	posX := elCenterX + float64(center.X) - (rect.Width / 2) + 3
	posY := elCenterY

	err = slide(el, posX, posY)
	if err != nil {
		return err
	}

	page.Race().ElementR(".geetest_radar_tip_content", "Succesful").MustHandle(func(e *rod.Element) {
		log.Println("Captcha passed")

	}).ElementR(".geetest_radar_tip_content", "Network failure").MustHandle(func(e *rod.Element) {
		el, err = page.Element(elRetrySlider)
		if err != nil {
			handleError(err)
			return
		}
		el.MustClick()

		el, err = page.Element(elSlider)
		if err != nil {
			handleError(err)
			return
		}

		err = slide(el, posX, posY)
		if err != nil {
			return
		}

	}).ElementR(".geetest_radar_tip_content", "Slide to complete the puzzle").MustHandle(func(e *rod.Element) {
		err = slide(el, posX, posY)
		if err != nil {
			return
		}

	}).MustDo()

	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {

	browser = rod.New().MustConnect()
	defer browser.MustClose()
	page = stealth.MustPage(browser)

	getCaptchaImages()

	saveDiffImage()

	center, err := findCenterImage()
	if err != nil {
		log.Println(err)
		return
	}

	err = moveCapchaSlider(center)
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("done")
	time.Sleep(time.Second * 500)
}

// go build -trimpath -ldflags="-s -w"
