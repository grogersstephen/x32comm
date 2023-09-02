### x32comm

X32comm is a utility to communicate with the Behringer X32 Audio Mixing Console via OSC messages.

[The OSC Specification](https://opensoundcontrol.stanford.edu/spec-1_0.html) by [Matthew Wright](https://github.com/matthewjameswright), one of the developers of OSC.

[UNOFFICIAL X32/M32 OSC REMOTE PROTOCOL](https://tostibroeders.nl/wp-content/uploads/2020/02/X32-OSC.pdf) by [Patrick-Gilles Maillot](https://github.com/pmaillot/)

## Installation

If Go is already installed:
```
go install github.com/grogersstephen/x32comm@latest
```

## Set up

Remote Addr

Local Port

## Usage

# Send a custom message with a float32 value
Send message "/ch/01/mix/fader" with float32 value ".5"
```
x32comm set --message /ch/01/mix/fader --float .5
```

# Send a custom message, and listen for a response message
Send message "/ch/01/mix/fader"
```
x32comm get --message /ch/01/mix/fader
```

# Get the value of a channel fader
Get the level of channel 5
```
x32comm getChFader 5
```

# Set the value of a channel fader in percentage
The percentage corresponds to the fader's physical position
Set channel 5 to 100% (+10dB)
```
x32comm getChFader 5 100
```
