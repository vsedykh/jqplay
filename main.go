package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"github.com/rivo/tview"
)

func main() {

	var parsedJSON map[string]any
	var timer *time.Timer
	const delay = 500 * time.Millisecond

	app := tview.NewApplication()

	taExp := tview.NewTextArea().
		SetWrap(false).
		SetPlaceholder("Enter expression here...")
	taExp.SetTitle("JQ expression").SetBorder(true)
	taJson := tview.NewTextArea().
		SetWrap(false).
		SetPlaceholder("Enter JSON here...")
	taJson.SetTitle("JSON").SetBorder(true)
	tvResult := tview.NewTextView().
		SetWrap(true)
	tvResult.SetTitle("Result").SetBorder(true)

	mainView := tview.NewGrid().
		SetRows(-1, -2).
		AddItem(taExp, 0, 0, 1, 1, 0, 0, true).
		AddItem(tvResult, 0, 1, 1, 1, 0, 0, false).
		AddItem(taJson, 1, 0, 1, 2, 0, 0, false)

	taExp.SetChangedFunc(func() {
		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(delay, func() {
			app.QueueUpdateDraw(func() {
				tvResult.SetText(runExpression(taExp.GetText(), &parsedJSON))
			})
		})
	})

	taJson.SetChangedFunc(func() {
		parseAndUpdate := func() {
			parsedJSON = make(map[string]any)
			err := json.Unmarshal([]byte(taJson.GetText()), &parsedJSON)
			if err != nil {
				tvResult.SetText(err.Error())
				return
			}
			tvResult.SetText(runExpression(taExp.GetText(), &parsedJSON))
		}

		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(delay, func() {
			app.QueueUpdateDraw(parseAndUpdate)
		})
	})

	if err := app.SetRoot(mainView, true).EnableMouse(true).EnablePaste(true).Run(); err != nil {
		panic(err)
	}
}

func runExpression(expression string, input *map[string]any) string {
	query, err := gojq.Parse(expression)
	if err != nil {
		return err.Error()
	}

	var result strings.Builder
	iter := query.Run(*input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			if err, ok := err.(*gojq.HaltError); ok && err.Value() == nil {
				break
			}
			return err.Error()
		}

		jsonData, err := json.Marshal(v)
		if err != nil {
			return err.Error()
		}
		result.WriteString(fmt.Sprintf("%s\n", string(jsonData)))
	}
	return result.String()
}
