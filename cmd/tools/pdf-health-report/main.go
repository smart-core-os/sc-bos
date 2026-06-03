// pdf-health-report generates a PDF health report from the CSV exports produced by the SC-BOS UI.
// It accepts a components CSV and a subsystem health CSV and writes a landscape A4 PDF.
//
// Usage:
//
//	pdf-health-report --components components_2026-6-3.csv --health subsystem-health_2026-6-3.csv \
//	                  [--logo /path/to/powered-by-smartcore.png] \
//	                  [--fonts-dir /path/to/poppins/] \
//	                  [--output report.pdf]
package main

import (
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	maroimage "github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/core/entity"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

//go:embed powered-by.png
var poweredByLogo []byte

// ------------------------------------------------------------------
// CLI flags
// ------------------------------------------------------------------

var (
	flagComponents string
	flagHealth     string
	flagOutput     string
	flagLogo       string
	flagFontsDir   string
)

func init() {
	flag.StringVar(&flagComponents, "components", "", "Path to components CSV file (required)")
	flag.StringVar(&flagHealth, "health", "", "Path to subsystem health CSV file (required)")
	flag.StringVar(&flagOutput, "output", "", "Output PDF path (default: health-report-YYYY-MM-DD.pdf)")
	flag.StringVar(&flagLogo, "logo", "", "Path to 'powered by smart core' logo PNG (optional)")
	flag.StringVar(&flagFontsDir, "fonts-dir", "", "Directory containing Poppins TTF files (optional; uses default font if omitted)")
}

// ------------------------------------------------------------------
// Smart Core colour palette  (from 19-cornwall-street/ui/esgdashboard/src/main.scss)
// ------------------------------------------------------------------

var (
	// Brand colours
	scBlack  = &props.Color{Red: 12, Green: 9, Blue: 33}   // #0C0921
	scNavy   = &props.Color{Red: 38, Green: 0, Blue: 77}   // #26004D
	scWhite  = &props.Color{Red: 248, Green: 244, Blue: 241} // #F8F4F1
	scViolet = &props.Color{Red: 127, Green: 0, Blue: 255} // #7F00FF

	// Table chrome
	colorTableHeaderBg = scNavy
	colorTableHeaderTx = scWhite
	colorSectionBg     = scNavy
	colorSectionTx     = scWhite
	colorBodyBg        = scWhite
	colorBodyText      = scBlack
	colorBorder        = &props.Color{Red: 80, Green: 0, Blue: 130} // muted violet border

	// Status row tints (light enough to keep text readable)
	colorRedBg   = &props.Color{Red: 255, Green: 213, Blue: 213}
	colorAmberBg = &props.Color{Red: 255, Green: 243, Blue: 205}
	colorGreenBg = &props.Color{Red: 213, Green: 242, Blue: 221}
	colorGreyBg  = &props.Color{Red: 235, Green: 235, Blue: 235}

	// Title bar accent strip (violet)
	_ = scViolet // referenced indirectly through titleAccentBg
	titleAccentBg = scViolet
)

// fontFamily is "Poppins" when fonts are loaded, otherwise the maroto default.
var fontFamily = ""

// ------------------------------------------------------------------
// Data types
// ------------------------------------------------------------------

type ComponentRow struct {
	Name, Address, Role, Connected            string
	Automations, Drivers, Systems             string
	CPUPct, MemPct, Uptime, LastRebootReason  string
}

type HealthRow struct {
	Subsystem, Group, Check, ItemID, ItemName string
	Status, Reliability, OfflineSince         string
	LastError, Faults                         string
}

// ------------------------------------------------------------------
// main
// ------------------------------------------------------------------

func main() {
	flag.Parse()

	if flagComponents == "" && flagHealth == "" {
		fmt.Fprintln(os.Stderr, "error: at least one of --components or --health is required")
		flag.Usage()
		os.Exit(1)
	}
	if flagOutput == "" {
		flagOutput = "health-report-" + time.Now().Format("2006-01-02") + ".pdf"
	}

	var components []ComponentRow
	if flagComponents != "" {
		var err error
		components, err = parseComponentsCSV(flagComponents)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading components CSV: %v\n", err)
			os.Exit(1)
		}
	}

	var healthRows []HealthRow
	if flagHealth != "" {
		var err error
		healthRows, err = parseHealthCSV(flagHealth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading health CSV: %v\n", err)
			os.Exit(1)
		}
	}

	doc, err := buildPDF(components, healthRows)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating PDF: %v\n", err)
		os.Exit(1)
	}

	if err := doc.Save(flagOutput); err != nil {
		fmt.Fprintf(os.Stderr, "error saving PDF: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Report written to %s\n", flagOutput)
}

// ------------------------------------------------------------------
// CSV parsing
// ------------------------------------------------------------------

func parseComponentsCSV(path string) ([]ComponentRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}

	var rows []ComponentRow
	for _, r := range records[1:] {
		if len(r) < 11 {
			continue
		}
		rows = append(rows, ComponentRow{
			Name: r[0], Address: r[1], Role: r[2], Connected: r[3],
			Automations: r[4], Drivers: r[5], Systems: r[6],
			CPUPct: r[7], MemPct: r[8], Uptime: r[9], LastRebootReason: r[10],
		})
	}
	return rows, nil
}

func parseHealthCSV(path string) ([]HealthRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}

	var rows []HealthRow
	for _, r := range records[1:] {
		if len(r) < 10 {
			continue
		}
		rows = append(rows, HealthRow{
			Subsystem: r[0], Group: r[1], Check: r[2], ItemID: r[3], ItemName: r[4],
			Status: r[5], Reliability: r[6], OfflineSince: r[7],
			LastError: r[8], Faults: r[9],
		})
	}
	return rows, nil
}

// ------------------------------------------------------------------
// PDF generation
// ------------------------------------------------------------------

func buildPDF(components []ComponentRow, healthRows []HealthRow) (core.Document, error) {
	b := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithOrientation(orientation.Horizontal).
		WithLeftMargin(5).
		WithTopMargin(0).
		WithRightMargin(5).
		WithBottomMargin(5)

	if flagFontsDir != "" {
		fonts, err := loadPoppins(flagFontsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not load Poppins fonts (%v); using default font\n", err)
		} else {
			b = b.WithCustomFonts(fonts)
			fontFamily = "Poppins"
		}
	}

	m := maroto.New(b.Build())

	addReportTitle(m)
	addComponentsSection(m, components)
	addHealthSection(m, healthRows)

	return m.Generate()
}

// loadPoppins loads Poppins Regular and SemiBold from the given directory.
func loadPoppins(dir string) ([]*entity.CustomFont, error) {
	regular, err := os.ReadFile(filepath.Join(dir, "Poppins-Regular.ttf"))
	if err != nil {
		return nil, fmt.Errorf("Poppins-Regular.ttf: %w", err)
	}
	semibold, err := os.ReadFile(filepath.Join(dir, "Poppins-SemiBold.ttf"))
	if err != nil {
		return nil, fmt.Errorf("Poppins-SemiBold.ttf: %w", err)
	}
	return []*entity.CustomFont{
		{Family: "Poppins", Style: fontstyle.Normal, Bytes: regular},
		{Family: "Poppins", Style: fontstyle.Bold, Bytes: semibold},
	}, nil
}

// ------------------------------------------------------------------
// Logo
// ------------------------------------------------------------------

// buildLogoCol returns a 3-column logo cell. It uses the --logo file when provided,
// otherwise falls back to the embedded powered-by.png bundled with the binary.
func buildLogoCol() core.Col {
	rect := props.Rect{Center: true, Percent: 75}
	if flagLogo != "" {
		return col.New(3).WithStyle(&props.Cell{BackgroundColor: scNavy}).Add(
			maroimage.NewFromFile(flagLogo, rect),
		)
	}
	return col.New(3).WithStyle(&props.Cell{BackgroundColor: scNavy}).Add(
		maroimage.NewFromBytes(poweredByLogo, extension.Png, rect),
	)
}

// ------------------------------------------------------------------
// Title
// ------------------------------------------------------------------

func addReportTitle(m core.Maroto) {
	// Thin violet accent strip at the very top
	m.AddRows(
		row.New(2).Add(
			col.New(12).WithStyle(&props.Cell{BackgroundColor: titleAccentBg}),
		),
	)

	titleText := "SC-BOS Health Report  —  " + time.Now().Format("2 January 2006")

	logoCol := buildLogoCol()
	if logoCol != nil {
		// Left: title text (9 cols), right: logo (3 cols)
		m.AddRows(
			row.New(16).Add(
				col.New(9).WithStyle(&props.Cell{BackgroundColor: scNavy}).Add(
					text.New(titleText, props.Text{
						Family: fontFamily,
						Size:   13,
						Style:  fontstyle.Bold,
						Align:  align.Left,
						Top:    5,
						Left:   4,
						Color:  scWhite,
					}),
				),
				logoCol,
			),
		)
	} else {
		m.AddRows(
			row.New(14).Add(
				col.New(12).WithStyle(&props.Cell{BackgroundColor: scNavy}).Add(
					text.New(titleText, props.Text{
						Family: fontFamily,
						Size:   13,
						Style:  fontstyle.Bold,
						Align:  align.Center,
						Top:    4,
						Color:  scWhite,
					}),
				),
			),
		)
	}

	m.AddRows(spacerRow(3, scNavy))
}

// ------------------------------------------------------------------
// Components section
// ------------------------------------------------------------------

func addComponentsSection(m core.Maroto, rows []ComponentRow) {
	m.AddRows(sectionHeaderRow("Components"))
	m.AddRows(componentHeaderRow())
	for _, r := range rows {
		m.AddRows(componentDataRow(r))
	}
	m.AddRows(spacerRow(5, nil))
}

func componentHeaderRow() core.Row {
	headers := []struct {
		label string
		width int
	}{
		{"Name", 2}, {"Address", 2}, {"Role", 1}, {"Connected", 1},
		{"Automations", 1}, {"Drivers", 1}, {"Systems", 1},
		{"CPU %", 1}, {"Mem %", 1}, {"Uptime", 1},
	}
	cols := make([]core.Col, 0, len(headers))
	for _, h := range headers {
		cols = append(cols, headerCell(h.label, h.width))
	}
	return row.New(7).Add(cols...)
}

func componentDataRow(r ComponentRow) core.Row {
	bg := componentRowColor(r)
	cells := []struct {
		val   string
		width int
	}{
		{r.Name, 2}, {r.Address, 2}, {r.Role, 1}, {r.Connected, 1},
		{r.Automations, 1}, {r.Drivers, 1}, {r.Systems, 1},
		{r.CPUPct, 1}, {r.MemPct, 1}, {r.Uptime, 1},
	}
	cols := make([]core.Col, 0, len(cells))
	for _, c := range cells {
		cols = append(cols, dataCell(c.val, c.width, bg))
	}
	return row.New(6).Add(cols...)
}

func componentRowColor(r ComponentRow) *props.Color {
	cpu, err := strconv.ParseFloat(r.CPUPct, 64)
	if err != nil {
		return colorBodyBg
	}
	if cpu >= 80 {
		return colorRedBg
	}
	if cpu >= 60 {
		return colorAmberBg
	}
	return colorBodyBg
}

// ------------------------------------------------------------------
// Health section
// ------------------------------------------------------------------

func addHealthSection(m core.Maroto, rows []HealthRow) {
	m.AddRows(sectionHeaderRow("Subsystem Health"))
	m.AddRows(healthHeaderRow())
	for _, r := range rows {
		m.AddRows(healthDataRow(r))
	}
}

func healthHeaderRow() core.Row {
	headers := []struct {
		label string
		width int
	}{
		{"Subsystem", 1}, {"Group", 1}, {"Check", 2}, {"Item Name", 2},
		{"Status", 1}, {"Reliability", 1}, {"Offline Since", 1},
		{"Last Error", 2}, {"Faults", 1},
	}
	cols := make([]core.Col, 0, len(headers))
	for _, h := range headers {
		cols = append(cols, headerCell(h.label, h.width))
	}
	return row.New(7).Add(cols...)
}

func healthDataRow(r HealthRow) core.Row {
	bg := healthRowColor(r)
	cells := []struct {
		val   string
		width int
	}{
		{r.Subsystem, 1}, {r.Group, 1}, {r.Check, 2}, {r.ItemName, 2},
		{r.Status, 1}, {r.Reliability, 1}, {r.OfflineSince, 1},
		{r.LastError, 2}, {r.Faults, 1},
	}
	cols := make([]core.Col, 0, len(cells))
	for _, c := range cells {
		cols = append(cols, dataCell(c.val, c.width, bg))
	}
	return row.New(6).Add(cols...)
}

func healthRowColor(r HealthRow) *props.Color {
	switch r.Status {
	case "Fault":
		return colorRedBg
	case "Degraded":
		return colorAmberBg
	case "OK":
		return colorGreenBg
	case "No data":
		return colorGreyBg
	default:
		return colorBodyBg
	}
}

// ------------------------------------------------------------------
// Shared helpers
// ------------------------------------------------------------------

func sectionHeaderRow(title string) core.Row {
	return row.New(8).Add(
		col.New(12).WithStyle(&props.Cell{BackgroundColor: colorSectionBg}).Add(
			text.New(title, props.Text{
				Family: fontFamily,
				Size:   10,
				Style:  fontstyle.Bold,
				Align:  align.Left,
				Top:    2,
				Left:   3,
				Color:  colorSectionTx,
			}),
		),
	)
}

func headerCell(label string, width int) core.Col {
	return col.New(width).WithStyle(&props.Cell{
		BackgroundColor: colorTableHeaderBg,
		BorderType:      border.Full,
		BorderColor:     colorBorder,
	}).Add(
		text.New(label, props.Text{
			Family: fontFamily,
			Size:   8,
			Style:  fontstyle.Bold,
			Align:  align.Left,
			Top:    1,
			Left:   1,
			Color:  colorTableHeaderTx,
		}),
	)
}

func dataCell(value string, width int, bg *props.Color) core.Col {
	style := &props.Cell{
		BorderType:      border.Full,
		BorderColor:     colorBorder,
		BackgroundColor: bg,
	}
	return col.New(width).WithStyle(style).Add(
		text.New(value, props.Text{
			Family: fontFamily,
			Size:   7,
			Align:  align.Left,
			Top:    1,
			Left:   1,
			Color:  colorBodyText,
		}),
	)
}

func spacerRow(height float64, bg *props.Color) core.Row {
	style := &props.Cell{}
	if bg != nil {
		style.BackgroundColor = bg
	}
	return row.New(height).Add(col.New(12).WithStyle(style))
}
