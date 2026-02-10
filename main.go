package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.design/x/clipboard"

	"github.com/gdamore/tcell/v2"
	"github.com/itchyny/gojq"
	"github.com/rivo/tview"
)

func main() {

	parsedJSON := map[string]any{}
	var timer *time.Timer
	const delay = 500 * time.Millisecond
	var taJsonWrap bool
	var taExpWrap bool
	var tvResultWrap bool

	app := tview.NewApplication()

	taExp := tview.NewTextArea().
		SetWrap(false).
		SetPlaceholder("Enter expression here...")
	taExp.SetTitle("JQ expression (^E)").SetBorder(true)
	taJson := tview.NewTextArea().
		SetWrap(false).
		SetPlaceholder("Enter JSON here...")
	taJson.SetTitle("JSON (^J)").SetBorder(true)
	tvResult := tview.NewTextView().
		SetWrap(true)
	tvResult.SetTitle("Result (^R)").SetBorder(true)
	tvHelpInfo := tview.NewTextView().
		SetText("Exit: ctrl+q | Copy: ^c | Paste: ^v | Select all: ^a | Minify JSON: ^m | Format JSON: ^f | Wrap: ^w")

	taJson.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlF:
			var formattedJson bytes.Buffer
			err := json.Indent(&formattedJson, []byte(taJson.GetText()), "", "  ")
			if err != nil {
				return nil
			}
			taJson.SetText(formattedJson.String(), false)
		case tcell.KeyCtrlM:
			var miniJson bytes.Buffer
			err := json.Compact(&miniJson, []byte(taJson.GetText()))
			if err != nil {
				return nil
			}
			taJson.SetText(miniJson.String(), false)
		case tcell.KeyCtrlW:
			taJsonWrap = !taJsonWrap
			taJson.SetWrap(!taJsonWrap)
			return nil
		}

		return event
	})

	taExp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlW:
			taExpWrap = !taExpWrap
			taExp.SetWrap(!taExpWrap)
			return nil
		}

		return event
	})

	tvResult.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlW:
			tvResultWrap = !tvResultWrap
			tvResult.SetWrap(!tvResultWrap)
			return nil
		}

		return event
	})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlE:
			app.SetFocus(taExp) // Set focus to the input field
		case tcell.KeyCtrlJ:
			app.SetFocus(taJson) // Set focus to the input field
		case tcell.KeyCtrlR:
			app.SetFocus(tvResult) // Set focus to the input field
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		case tcell.KeyCtrlC:
			el := app.GetFocus()
			if ta, ok := el.(*tview.TextArea); ok {
				text, _, _ := ta.GetSelection()
				clipboard.Write(clipboard.FmtText, []byte(text))
				return nil
			}
			if tv, ok := el.(*tview.TextView); ok {
				text := tv.GetText(true)
				clipboard.Write(clipboard.FmtText, []byte(text))
				return nil
			}
			return nil
		case tcell.KeyCtrlA:
			el := app.GetFocus()
			if ta, ok := el.(*tview.TextArea); ok {
				ta.Select(0, ta.GetTextLength())
				return nil
			}
			return event
		}
		return event
	})

	mainView := tview.NewGrid().
		SetRows(-1, -2, 1).
		AddItem(taExp, 0, 0, 1, 1, 0, 0, false).
		AddItem(tvResult, 0, 1, 1, 1, 0, 0, false).
		AddItem(taJson, 1, 0, 1, 2, 0, 0, true).
		AddItem(tvHelpInfo, 2, 0, 1, 2, 0, 0, false)

	taExp.SetChangedFunc(func() {
		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(delay, func() {
			app.QueueUpdateDraw(func() {
				tvResult.SetText(runExpression(taExp.GetText(), parsedJSON))
			})
		})
	})

	taExp.SetFocusFunc(func() {
		tvHelpInfo.SetText("Exit: ctrl+q | Copy: ^c | Paste: ^v | Select all: ^a | Wrap: ^w")
	})

	taJson.SetChangedFunc(func() {
		parseAndUpdate := func() {
			parsedJSON = make(map[string]any)
			err := json.Unmarshal([]byte(taJson.GetText()), &parsedJSON)
			if err != nil {
				tvResult.SetText(fmt.Sprintf("invalid json: %s", err.Error()))
				return
			}
			tvResult.SetText(runExpression(taExp.GetText(), parsedJSON))
		}

		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(delay, func() {
			app.QueueUpdateDraw(parseAndUpdate)
		})
	})

	taJson.SetFocusFunc(func() {
		tvHelpInfo.SetText("Exit: ctrl+q | Copy: ^c | Paste: ^v | Select all: ^a | Minify JSON: ^m | Format JSON: ^f | Wrap: ^w")
	})

	tvResult.SetFocusFunc(func() {
		tvHelpInfo.SetText("Exit: ctrl+q | Copy: ^c | Minify JSON: ^m | Format JSON: ^f | Wrap: ^w")
	})

	if err := app.SetRoot(mainView, true).EnableMouse(true).EnablePaste(true).SetTitle("JQplay").Run(); err != nil {
		panic(err)
	}
}

func runExpression(expression string, input map[string]any) string {
	query, err := gojq.Parse(expression)
	if err != nil {
		return fmt.Sprintf("invalid jq expression: %s", err.Error())
	}

	var result strings.Builder
	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return fmt.Sprintf("invalid jq expression: %s", err.Error())
		}

		jsonData, err := json.Marshal(v)
		if err != nil {
			return err.Error()
		}
		result.WriteString(fmt.Sprintf("%s\n", string(jsonData)))
	}
	return result.String()
}
