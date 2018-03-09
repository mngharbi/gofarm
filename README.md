# Gofarm  ![Travis CI Build Status](https://api.travis-ci.org/mngharbi/gofarm.svg?branch=master) [![Coverage](https://codecov.io/gh/mngharbi/gofarm/branch/master/graph/badge.svg)](https://codecov.io/gh/mngharbi/gofarm) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/mngharbi/gofarm/master/LICENSE)

Gofarm is a generic server/worker implementation for Go.

## Overview

Gofarm allows you to have a generic server (one instance).

All you need to do is implement server callbacks as defined [here](https://github.com/mngharbi/gofarm/blob/master/types.go).

It provides a thread-safe API to start, shutdown, force shutdown, or make generic requests.

Make requests to the server simply consists of passing a generic Request, and receiving a channel for a generic response. The channel will either have a value when ready or close appropriately during force shutdown.

Gofarm will also make calls to user-defined functions during startup, shutdown, and for processing requests.

## Installation

With a healthy Go Language installed, simply run `go get github.com/mngharbi/gofarm`
