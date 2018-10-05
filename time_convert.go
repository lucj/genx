package main

import (
        "strconv"
       )

func GetSeconds(ts string) int {

        // Get time indicator (d: day, h: hour, m: minute)
        ind := string(ts[len(ts)-1])

        // Get corresponding number of seconds
        sec := 0
        switch ind {
        case "d":
	        sec = 24 * 60 * 60
        case "h":
	        sec = 60 * 60
        case "m":
	        sec = 60
        case "s":
	        sec = 1
        default:
	    panic("unrecognized escape character")
        } 

        // Get number of time the indicator is used
        nbr, _ := strconv.Atoi(ts[:len(ts)-1])

        // Get duration in seconds
        return nbr * sec
}
