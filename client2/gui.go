//go:build gio
// +build gio

package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"path/filepath"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/atotto/clipboard"
)

type GioUI struct {
	client          *Client
	theme           *material.Theme
	statusText      string
	currentFiles    []string
	selectedFile    int
	uploadBtn       widget.Clickable
	sendBtn         widget.Clickable
	refreshBtn      widget.Clickable
	downloadBtn     widget.Clickable
	copyUUIDBtn     widget.Clickable
	fileList        widget.List
	fileListButtons []widget.Clickable
	uuidEntry       widget.Editor
	filePathEntry   widget.Editor
	showInputPanel  bool
	inputMode       string // "upload" or "send"
	submitBtn       widget.Clickable
	cancelBtn       widget.Clickable
}

func NewGioUI(client *Client) *GioUI {
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	return &GioUI{
		client:       client,
		theme:        theme,
		statusText:   "✓ Connected to server",
		selectedFile: -1,
		fileList: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		uuidEntry: widget.Editor{
			SingleLine: true,
			Submit:     true,
		},
		filePathEntry: widget.Editor{
			SingleLine: true,
			Submit:     true,
		},
		showInputPanel: false,
		inputMode:      "",
	}
}

func (ui *GioUI) refreshFiles() {
	files, err := ui.client.ListFiles()
	if err != nil {
		ui.statusText = "❌ Failed to list files: " + err.Error()
		return
	}
	ui.currentFiles = files

	// Ensure we have enough clickable widgets for all files
	for len(ui.fileListButtons) < len(files) {
		ui.fileListButtons = append(ui.fileListButtons, widget.Clickable{})
	}

	ui.statusText = fmt.Sprintf("✓ %d files available", len(files))
}

func (ui *GioUI) Run(w *app.Window) error {
	var ops op.Ops

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Handle button clicks
			if ui.uploadBtn.Clicked(gtx) {
				// Open file picker
				go func() {
					filename, err := openFileDialog("Select file to upload")
					if err == nil && filename != "" {
						ui.statusText = "⏳ Uploading..."
						err = putFile(filename, ui.client.GetConnection(), 1024)
						if err != nil {
							ui.statusText = "❌ Upload failed: " + err.Error()
						} else {
							ui.statusText = "✓ File uploaded successfully!"
							ui.refreshFiles()
						}
						w.Invalidate()
					}
				}()
			}

			if ui.sendBtn.Clicked(gtx) {
				ui.showInputPanel = true
				ui.inputMode = "send"
				ui.filePathEntry.SetText("")
				ui.uuidEntry.SetText("")
			}

			if ui.refreshBtn.Clicked(gtx) {
				ui.refreshFiles()
			}

			if ui.copyUUIDBtn.Clicked(gtx) {
				err := clipboard.WriteAll(ui.client.GetUID())
				if err != nil {
					ui.statusText = "❌ Failed to copy UUID"
				} else {
					ui.statusText = "✓ UUID copied to clipboard!"
				}
			}

			if ui.downloadBtn.Clicked(gtx) {
				if ui.selectedFile >= 0 && ui.selectedFile < len(ui.currentFiles) {
					filename := ui.currentFiles[ui.selectedFile]
					savePath := "downloaded_" + filepath.Base(filename)

					ui.statusText = "⏳ Downloading..."
					err := ui.client.DownloadFile(filename, savePath)
					if err != nil {
						ui.statusText = "❌ Download failed: " + err.Error()
					} else {
						ui.statusText = fmt.Sprintf("✓ Downloaded as %s", savePath)
						ui.refreshFiles()
					}
				} else {
					ui.statusText = "⚠️ Please select a file first"
				}
			}

			// Handle submit button (for send mode - needs file selection)
			if ui.submitBtn.Clicked(gtx) {
				if ui.inputMode == "send" {
					targetUUID := ui.uuidEntry.Text()
					if targetUUID == "" {
						ui.statusText = "⚠️ Please enter a target UUID"
					} else {
						// Open file picker
						go func() {
							filename, err := openFileDialog("Select file to send")
							if err == nil && filename != "" {
								ui.statusText = "⏳ Sending file..."
								err = ui.client.SendFileToUUID(filename, targetUUID)
								if err != nil {
									ui.statusText = "❌ Send failed: " + err.Error()
								} else {
									ui.statusText = fmt.Sprintf("✓ File sent to %s", targetUUID)
									ui.showInputPanel = false
								}
								w.Invalidate()
							}
						}()
					}
				}
			}

			// Handle cancel button
			if ui.cancelBtn.Clicked(gtx) {
				ui.showInputPanel = false
			}

			// Draw the UI
			ui.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}

func (ui *GioUI) Layout(gtx layout.Context) layout.Dimensions {
	// If input panel is shown, show overlay
	if ui.showInputPanel {
		return ui.layoutInputPanel(gtx)
	}

	// Main container with padding
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			// Header section
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						title := material.H6(ui.theme, "fsend - File Sharing")
						title.Color = color.NRGBA{R: 63, G: 81, B: 181, A: 255}
						return title.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Body2(ui.theme, ui.statusText)
						return label.Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								label := material.Caption(ui.theme, fmt.Sprintf("Your UUID: %s", ui.client.GetUID()))
								label.Color = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
								return label.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.copyUUIDBtn, "📋 Copy")
								btn.Background = color.NRGBA{R: 100, G: 100, B: 100, A: 255}
								btn.TextSize = unit.Sp(12)
								return btn.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
				)
			}),

			// File list section
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						label := material.Body1(ui.theme, "📁 Your Files:")
						label.Font.Weight = 500
						return label.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if len(ui.currentFiles) == 0 {
							label := material.Body2(ui.theme, "(No files)")
							label.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
							return label.Layout(gtx)
						}

						return material.List(ui.theme, &ui.fileList).Layout(gtx, len(ui.currentFiles), func(gtx layout.Context, index int) layout.Dimensions {
							// Make items clickable
							if index >= len(ui.fileListButtons) {
								return layout.Dimensions{}
							}

							btn := material.ButtonLayoutStyle{
								Background:   color.NRGBA{R: 240, G: 240, B: 240, A: 255},
								CornerRadius: unit.Dp(4),
								Button:       &ui.fileListButtons[index],
							}

							if index == ui.selectedFile {
								btn.Background = color.NRGBA{R: 200, G: 220, B: 255, A: 255}
							}

							if ui.fileListButtons[index].Clicked(gtx) {
								ui.selectedFile = index
							}

							return btn.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									label := material.Body2(ui.theme, ui.currentFiles[index])
									return label.Layout(gtx)
								})
							})
						})
					}),
				)
			}),

			// Button section
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.uploadBtn, "📤 Upload")
								btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255}
								return btn.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.sendBtn, "📨 Send to UUID")
								btn.Background = color.NRGBA{R: 33, G: 150, B: 243, A: 255}
								return btn.Layout(gtx)
							}),
						)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.refreshBtn, "🔄 Refresh")
								btn.Background = color.NRGBA{R: 158, G: 158, B: 158, A: 255}
								return btn.Layout(gtx)
							}),
							layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								btn := material.Button(ui.theme, &ui.downloadBtn, "⬇️ Download")
								btn.Background = color.NRGBA{R: 255, G: 152, B: 0, A: 255}
								return btn.Layout(gtx)
							}),
						)
					}),
				)
			}),
		)
	})
}

func (ui *GioUI) layoutInputPanel(gtx layout.Context) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// Create a card-like panel
		gtx.Constraints.Max.X = gtx.Dp(unit.Dp(400))

		return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					title := "Upload File"
					if ui.inputMode == "send" {
						title = "Send File to UUID"
					}
					label := material.H6(ui.theme, title)
					return label.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),

				// UUID input (only for send mode)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if ui.inputMode != "send" {
						return layout.Dimensions{}
					}
					return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							label := material.Body2(ui.theme, "Target UUID:")
							return label.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							editor := material.Editor(ui.theme, &ui.uuidEntry, "Enter target UUID...")
							editor.Color = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
							return editor.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(20)}.Layout),

				// Buttons
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Spacing: layout.SpaceEvenly}.Layout(gtx,
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							btnText := "Select File"
							if ui.inputMode == "send" {
								btnText = "Next"
							}
							btn := material.Button(ui.theme, &ui.submitBtn, btnText)
							btn.Background = color.NRGBA{R: 76, G: 175, B: 80, A: 255}
							return btn.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.cancelBtn, "Cancel")
							btn.Background = color.NRGBA{R: 244, G: 67, B: 54, A: 255}
							return btn.Layout(gtx)
						}),
					)
				}),
			)
		})
	})
}

func RunGUI() {
	// Connect to server
	client, err := NewClient("localhost:3002")
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	err = client.Connect()
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}
	defer client.Close()

	// Create window
	go func() {
		w := new(app.Window)
		w.Option(app.Title("fsend - File Sharing"))
		w.Option(app.Size(unit.Dp(600), unit.Dp(500)))

		ui := NewGioUI(client)
		ui.refreshFiles()

		if err := ui.Run(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	app.Main()
}
