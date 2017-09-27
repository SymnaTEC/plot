/*
 SymnaTEC plot - Displays muscle activity measured using a Raspberry Pi
 Copyright (c) Dorian Stoll 2017
 Licensed under the Terms of the MIT License
 */

package main

import (
    "github.com/SymnaTEC/go-adcpi"
    "github.com/buger/goterm"
    "os"
    "fmt"
    "time"
    "bufio"
    "strings"
    "flag"
    "strconv"
    "math/rand"
)

/*
 This is the entry point of the application. When the program is run, this will be the first function that gets called.
 It is responsible for creating the connection to the muscle sensor and starting the plotting tools.
 */
func main() {

    // Clear the terminal
    goterm.Clear()

    // Load the settings from the command line
    LoadSettings()

    // Create a channel to connect the two threads, the data thread and the display thread
    channel := make(chan float64)

    // Start the background thread that reads the voltage data
    if Settings.Debug {
        go grabRandomData(channel)
    } else if Settings.Playback {
        go grabDataFromFile(channel)
    } else {
        go grabDataFromADCPI(channel)
    }

    // Receive the data from the background thread
    keys := []float64{}
    values := []float64{}
    x := 0
    for v := range channel {

        // Append the new values to the general collection
        keys = append(keys, float64(x) * Settings.Interval)
        values = append(values, v)

        // Prepare a Table for the last x values
        data := &goterm.DataTable{}
        data.AddColumn("Time")
        data.AddColumn("Voltage")

        // Add the last x values from the value arrays to the table
        i := min(len(keys), Settings.Scale)
        for i > 0 {
            data.AddRow(keys[len(keys)-i], values[len(values)-i])
            i--
        }

        // Move the cursor to the beginning so we clear the console
        goterm.MoveCursor(0, 0)

        // Create a new chart
        chart := goterm.NewLineChart(Settings.Width, Settings.Height)
        chart.Flags = goterm.DRAW_RELATIVE

        // Draw the table using the chart
        fmt.Println(chart.Draw(data))
        goterm.Flush()
        x++
    }
}

/*
 A small helper function to return the smaller number
 */
func min(x int, y int) int {
    if x < y {
        return x
    }
    return y
}

/*
 This function queries the ADCPi extension board, and writes the voltage readout into the channel between this
 function and the plotting logic
 */
func grabDataFromADCPI(channel chan float64) {

    // Connect to the ADCPi
    adc := adcpi.ADCPI(byte(Settings.Address), 18)

    // Create the CSV file
    csv,err := os.Create(Settings.File)
    if err != nil {
        panic(err)
    }
    csv.WriteString("Time;Voltage")
    defer csv.Close()
    defer close(channel)

    // Counter
    x := 0
    voltage := float64(0)

    // Create an infinite loop
    for true {
        voltage = adc.ReadVoltage(byte(Settings.Channel))
        channel <- voltage
        csv.WriteString(fmt.Sprintf("\n%f;%f", float64(x) * Settings.Interval, voltage))
        x++
        // Converts our decimal value in seconds to an integer value in nanoseconds
        time.Sleep(time.Duration(Settings.Interval * 1000 * 1000 * 1000))
    }
}

/*
 This function queries a previously created file, and writes the voltage readout into the channel between this
 function and the plotting logic
 */
func grabDataFromFile(channel chan float64) {

    // Load the file
    csv,err := os.Open(Settings.File)
    if err != nil {
        panic(err)
    }
    scan := bufio.NewReader(csv)
    defer close(channel)

    // Counter
    x := 0
    voltage := float64(0)
    line := ""
    scan.ReadString(10) // Skip CSV declaration

    // Create an infinite loop
    for true {
        line, err = scan.ReadString(10)
        if line != "" {
            voltage, err = strconv.ParseFloat(strings.Replace(strings.Split(line, ";")[1],
                "\n", "", -1), 64)
            if err != nil {
                panic(err)
            }
            channel <- voltage
            x++
        }
        // Converts our decimal value in seconds to an integer value in nanoseconds
        time.Sleep(time.Duration(Settings.Interval * 1000 * 1000 * 1000))
    }
}

/*
 This function generates random voltage data and writes it into the channel between this function
 and the plotting logic
 */
func grabRandomData(channel chan float64) {

    // Create an infinite loop
    for true {

        // Random value between 0 and 5
        channel <- rand.Float64() * 5

        // Converts our decimal value in seconds to an integer value in nanoseconds
        time.Sleep(time.Duration(Settings.Interval * 1000 * 1000 * 1000))
    }
}

/*
 A type that stores all settings. These settings are loaded through command line arguments.
 Example:
    $ plot --file=data.csv --address=0x68 --channel=1
    $ plot --file=data.csv --playback
 */
type SettingsData struct {

    /*
     The file where the data from the muscle sensor will be stored. It should end with .csv, but any file extension
     is acceptable. If playback mode is enabled, the program will not store data in the file but load it.
     */
    File string

    /*
     The I2C address of the interface we are connecting to. The default setting is 0x68 (so 104 in decimal notation).
     */
    Address int

    /*
     The channel of the analog pin where the muscle sensor is connected.
     */
    Channel int

    /*
     Whether the playback mode should be enabled. In playback mode, the application won't connect to the muscle sensor
     but load existing data and display it again.
     */
    Playback bool

    /*
     The amount of seconds that passes between two measurements
     */
    Interval float64

    /*
     In debug mode, the program generates random data and plots that
     */
    Debug bool

    /*
     Defines how many values should get plotted at the same time
     */
    Scale int

    /*
     The width of the command line plot
     */
    Width int

    /*
     The height of the command line plot
     */
    Height int
}

/*
 The Instance of the Settings Storage
 */
var Settings SettingsData

func LoadSettings() {
    Settings = SettingsData{}
    flag.StringVar(&(Settings.File), "file", "", "The file where the data from the muscle " +
        "sensor will be stored. If playback mode is enabled, the program will not store data in the file but load it.")
    flag.IntVar(&(Settings.Address), "address", 0x68, "The I2C address of the interface we " +
        "are connecting to.")
    flag.IntVar(&(Settings.Channel), "channel", 1, "The channel of the analog pin where the " +
        "muscle sensor is connected.")
    flag.BoolVar(&(Settings.Playback), "playback", false, "Whether the playback mode should be " +
        "enabled. In playback mode, the applications won't connect to the muscle sensor but load existing data and " +
        "display it again.")
    flag.Float64Var(&(Settings.Interval), "interval", 0.1, "The amount of seconds that passes " +
        "between two measurements")
    flag.BoolVar(&(Settings.Debug), "debug", false, "In debug mode, the program generates " +
        "random data and plots that")
    flag.IntVar(&(Settings.Scale), "scale", 20, "Defines how many values should get plotted " +
        "at the same time")
    flag.IntVar(&(Settings.Width), "width", goterm.Width(), "The width of the command line plot")
    flag.IntVar(&(Settings.Height), "height", goterm.Height(), "The height of the command line plot")
    flag.Parse()
}

