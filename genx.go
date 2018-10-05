package main

import (
        "time"
        "flag"
        "fmt"
        "math"
       )

func main() {


        // Define global flags
        typePtr := flag.String("type", "cos", "type of curve")
        durationPtr := flag.String("duration", "1d", "duration")
        stepPtr := flag.String("step", "1h", "step")
        // realtimePtr := flag.Bool("realtime", false, "realtime generation")

        // Special parameters for linear curve
        linearFirstValue := flag.Float64("first", 0, "first value")
        linearLastValue := flag.Float64("last", 1, "last value")

        // Special parameters for cosinus curve
        cosMinValue := flag.Float64("min", 10, "min value")
        cosMaxValue := flag.Float64("max", 25, "max value")
        cosPeriod := flag.String("period", "1d", "period")

        // Parse parameters
        flag.Parse()

        // Get number of items to generate
        durationSeconds := GetSeconds(*durationPtr)
        stepSeconds := GetSeconds(*stepPtr)
        itemNbr := durationSeconds / stepSeconds

        // Get start date
        start := time.Now().Unix()

        // Default function is basic x => x
        fn := func(x float64) float64 { return x }

        // Check curve type
        // TODO: move curve generation in dedicated file
        switch *typePtr {
        case "linear":
                // Get linear related parameters
                first := *linearFirstValue
                last := *linearLastValue

                // Build function : y = A.(x-B)+C
                fn = func(x float64) float64 { 
                  A := (last - first) / float64(durationSeconds)
                  B := float64(start)
                  C := first
                  return A * ( x - B) + C
                }

        case "cos":
                // Get cosinus related parameters
                min := *cosMinValue
                max := *cosMaxValue
                periodSeconds := GetSeconds(*cosPeriod)

                // Build function : y = A.cos(B(x-C))+D
                fn = func(x float64) float64 { 
                  A := (max - min) / 2
                  B := float64(2 * math.Pi) / float64(periodSeconds)
                  C := 0.0
                  D := min + A
                  return A * math.Cos(B * (x - C)) + D
                }

        default:
                panic("uncorrect function type")
        }

        // Generate items
        for i := 0; i < itemNbr; i++ {
                ts := start + int64(i * stepSeconds)
                fmt.Printf("%d %.2f\n", ts, fn(float64(ts)))
	}
}
