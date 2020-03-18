[![Build Status](https://travis-ci.com/gol4ng/signal.svg?branch=master)](https://travis-ci.com/gol4ng/signal)
[![Maintainability](https://api.codeclimate.com/v1/badges/67b0678ef69a37037689/maintainability)](https://codeclimate.com/github/gol4ng/signal/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/67b0678ef69a37037689/test_coverage)](https://codeclimate.com/github/gol4ng/signal/test_coverage)

# signal
This repository provides helpers with signals

## Subscriber
Signal subscriber that allows you to attach a callback to an `os.Signal` notification.

Useful to react to any os.Signal.

It returns an `unsubscribe` function that can gracefully stop some http server and clean allocated object
