# TeeTimeFinder

![Demo](demo.gif)

A command-line tool to find and book golf tee times from MiClub and Quick18 websites.

MiClub: https://www.miclub.com.au/cms/

Sagacity/Quick18: https://www.sagacitygolf.com/

TeeTimeFinder aims to solve the problem of manually going through multiple course websites to find a tee time available. You can easily search with TeeTimeFinder all the local courses in your area to find a tee time available. Once you have found a tee time, you will be given the URL to complete the booking.

## Features
- Search tee times at golf courses all at once
- Filter results by date, time and players
- Supports both MiClub and Quick18 booking platforms (The two most popular online booking platforms in Australia)
- Interactive prompts and command-line flags 

## Installation
Requires `go` to be installed.

1. Clone the repo:

``` shell
git clone https://github.com/Ay1tsMe/teetimefinder.git
```

2. Build from source:

``` shell
cd teetimefinder
go build -o TeeTimeFinder
```

3. Install to PATH

``` shell
go install
```

4. Install autocompletion suggestions:

``` shell
# Choose one based on your shell
TeeTimeFinder completion zsh > ~/.zsh/completions/_TeeTimeFinder
TeeTimeFinder completion bash > ~/.bash/completions/_TeeTimeFinder
TeeTimeFinder completion fish > ~/.fish/completions/_TeeTimeFinder

# Navigate to your shell .rc file and add the following line (This is different for each shell)
echo "fpath=(~/.zsh/completions $fpath)" >> ~/.zshrc
echo "autoload -U compinit && compinit" >> ~/.zshrc

# Then reload
source ~/.zshrc
```

## Configuration
Before first use, configure which golf courses to search from:

``` shell
TeeTimeFinder config
```
Follow the prompts to add:
- Course name
- Booking URL e.g. (https://maylandsembleton.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000) 
- Website type (MiClub/Quick18)

You can view your configured courses here:

``` shell
TeeTimeFinder config show
```

## Usage

### Basic Search

``` shell
TeeTimeFinder
```
- Interactive prompts for date/time/player filters
- Searches all configured courses

### Advanced Search with Flags

The following searches for Royal Perth and Royal Fremantle for a tee time on the 17/08/2024 at 9am for 2 or more players.

``` shell
TeeTimeFinder -d 17-08-2024 -t 09:00 -s 2 -c "Royal Perth Golf Club" -c "Royal Fremantle Golf Club"
```

### Commands and Flags
Main Commands
| Flag          | Description                                                            | Example       |
|---------------|------------------------------------------------------------------------|---------------|
| -d, --date    | Search date (DD-MM-YYYY)                                               | -d 24-06-2025 |
| -t, --time    | Centre time for 2hr window (±1 hour)                                   | -t 14:30      |
| -s, --spots   | Minimum available player spots (1-4)                                   | -s 3          |
| -c, --courses | Specify particular courses to search                                   | -c "Course"   |
| -v, --verbose | Enable verbose debug output (debug.log file found in config directory) |               |

Configuration Commands

``` shell
# Overwrite course config
TeeTimeFinder config [-o|--overwrite]

# List configured courses
TeeTimeFinder config show
```

## Examples
1. Find Saturday morning times with 4 spots:

``` shell
TeeTimeFinder -d 22-02-2025 -t 08:00 -s 4
```

2. Find Friday morning times at Hartfield Golf Club

``` shell
TeeTimeFinder -d 21-02-2025 -t 07:00 -c "Hartfield Golf Club"
```

3. Search for all tee times on Thursday

``` shell
TeeTimeFinder -d 06-02-2025
```

## Example Config
The following is an example config file for TeeTimeFinder. Use this as a reference for what type of URLs are needed for TeeTimeFinder to search.

``` shell
Secret Harbour Golf Club,https://secretharbour.miclub.com.au/guests/bookings/ViewPublicCalendar.msp,miclub
Kennedy Bay Golf Club,https://kennedybay.miclub.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000,miclub
The Springs Golf Course,https://springs.quick18.com/teetimes/searchmatrix,Quick18
Hamersley Golf Course,https://hamersley.quick18.com/teetimes/searchmatrix,Quick18
Hartfield Golf Club,https://www.hartfieldgolf.com.au/guests/bookings/ViewPublicCalendar.msp?booking_resource_id=3000000,Miclub
```

## Running Tests
There are multiple tests files in folders `cmd` and `pkg`. Before contributing code, make sure that your code passes all tests.

To run the tests, run the following in the project root:

``` shell
# Test cmd files
cd cmd
go test
```

``` shell
# Test pkg files
cd pkg
go test ./...
```
### Running tests against online URL's
You can also run tests against the live websites which TeeTimeFinder searches. 

To run the tests, run the following in the project root:

``` shell
# Run tests against live websites
cd pkg/miclub
go test -args -online
```

## Licence
This project is licensed under the MIT Licence. See the LICENCE file for more information.
