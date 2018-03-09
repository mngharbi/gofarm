# Gofarm  ![Travis CI Build Status](https://api.travis-ci.org/mngharbi/gofarm.svg?branch=master) [![Coverage](https://codecov.io/gh/mngharbi/gofarm/branch/master/graph/badge.svg)](https://codecov.io/gh/mngharbi/gofarm) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/mngharbi/gofarm/master/LICENSE)

Gofarm is a generic server/worker implementation for Go.

## Overview

Gofarm allows you to have a generic server (one instance).

All you need to do is to implement a server as defined [here](https://github.com/mngharbi/gofarm/blob/master/types.go).

It provides a thread-safe API to start, shutdown, force shutdown, or make generic requests with a channel for a generic response as a return value.

Gofarm will also make calls to user-defined functions during startup, shutdown, and for processing requests.

## Installation

With a healthy Go Language installed, simply run `go get github.com/mngharbi/gofarm`
