package main

import (
        "time"
        "flag"
        "fmt"
       )

func main() {


        // Define global flags
        typePtr := flag.String("type", "cos", "type of curve")
        durationPtr := flag.String("duration", "1d", "duration of the generation")
        stepPtr := flag.String("step", "1h", "step / sampling period")
        // realtimePtr := flag.Bool("realtime", false, "realtime generation")

        // Special parameters for linear curve
        linearFirstValue := flag.Float64("first", 0, "first value for linear type")
        linearLastValue := flag.Float64("last", 1, "last value for linear type")

        // Special parameters for cosinus curve
        cosMinValue := flag.Float64("min", 10, "min value for cos type")
        cosMaxValue := flag.Float64("max", 25, "max value for cos type")
        cosPeriod := flag.String("period", "1d", "period for cos type")

        // Parse parameters
        flag.Parse()

        // Get number of items to generate
        durationSeconds := GetSeconds(*durationPtr)
        stepSeconds := GetSeconds(*stepPtr)
        itemNbr := durationSeconds / stepSeconds

        // Default function is basic x => 1
        fn := func(x float64) float64 { return 1 }

        // Get start time
        start := time.Now().Unix()

        // Check curve type
        switch *typePtr {
        case "linear":
            fn = GetLinear(*linearFirstValue, *linearLastValue, start, durationSeconds)
        case "cos":
            fn = GetCosinus(*cosMinValue, *cosMaxValue, *cosPeriod)
        case "log":
            fn = GetLog(start)
        case "exp":
            fn = GetExp(start, durationSeconds)
        default:
            panic("uncorrect function type")
        }

        // Generate items
        for i := 0; i < itemNbr; i++ {
                ts := start + int64(i * stepSeconds)
                fmt.Printf("%d,%.2f\n", ts, fn(float64(ts)))
	}
}
