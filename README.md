# TeeTimeFinder

A command-line tool to find and book golf tee times from MiClub and Quick18 websites.

MiClub: https://www.miclub.com.au/cms/
Sagacity/Quick18: https://www.sagacitygolf.com/

TeeTimeFinder aims to solve the problem of manually going through multiple course websites to find a tee time available. You can easily search with TeeTimeFinder all the local courses in your area to find a tee time available.

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
