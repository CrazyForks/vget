package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	_ "embed"
)

//go:embed logo.png
var logoPNG []byte

// forcedVariant wraps a theme and forces a specific variant
type forcedVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

func main() {
	a := app.New()
	isDark := true
	a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	w := a.NewWindow("VGet Downloader")

	// Logo
	logoResource := fyne.NewStaticResource("logo.png", logoPNG)
	logo := canvas.NewImageFromResource(logoResource)
	logo.SetMinSize(fyne.NewSize(32, 32))
	logo.FillMode = canvas.ImageFillContain

	// Header
	title := widget.NewLabel("VGet")
	title.TextStyle = fyne.TextStyle{Bold: true}

	themeBtn := widget.NewButton("☀", nil)
	themeBtn.OnTapped = func() {
		isDark = !isDark
		if isDark {
			a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
			themeBtn.SetText("☀")
		} else {
			a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantLight})
			themeBtn.SetText("🌙")
		}
	}

	headerLeft := container.NewHBox(logo, title)
	header := container.NewBorder(nil, nil, headerLeft, themeBtn)

	// URL input
	input := widget.NewEntry()
	input.SetPlaceHolder("Enter URL...")

	downloadBtn := widget.NewButton("Download", func() {
		// TODO: implement download
	})

	inputRow := container.NewBorder(nil, nil, nil, downloadBtn, input)

	// Layout
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewPadded(inputRow),
	)

	w.SetContent(container.New(layout.NewPaddedLayout(), content))
	w.Resize(fyne.NewSize(900, 600))
	w.ShowAndRun()
}
