package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/serg2014/shortener/internal/checker/osexit"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/staticcheck"
)

// Config — имя файла конфигурации.
const Config = `config.json`

// ConfigData описывает структуру файла конфигурации.
type ConfigData struct {
	Staticcheck []string
}

// TODO
// В staticlint используешь panic для обработки ошибок - лучше использовать log.Fatal
// или возвращать ошибки из отдельной функции run()
func main() {
	appfile, err := os.Executable()
	if err != nil {
		panic(err)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(appfile), Config))
	if err != nil {
		panic(err)
	}
	var cfg ConfigData
	if err = json.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}
	mychecks := []*analysis.Analyzer{
		printf.Analyzer,
		structtag.Analyzer,
		fieldalignment.Analyzer,
		waitgroup.Analyzer,
		bodyclose.Analyzer,
		osexit.Analyzer,
	}
	checks := make(map[string]bool)
	var checksPattern []string
	for _, v := range cfg.Staticcheck {
		pref, ok := strings.CutSuffix(v, "*")
		if ok {
			checksPattern = append(checksPattern, pref)
		} else {
			checks[v] = true
		}
	}

	// добавляем анализаторы из staticcheck, которые указаны в файле конфигурации
	for _, v := range staticcheck.Analyzers {
		if checks[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		} else {
			for _, pref := range checksPattern {
				if strings.HasPrefix(v.Analyzer.Name, pref) {
					mychecks = append(mychecks, v.Analyzer)
				}
			}
		}
	}
	multichecker.Main(
		mychecks...,
	)
}
