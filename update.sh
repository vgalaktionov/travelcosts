#!/bin/bash

echo "Building and setting executable flags..."
go build && chmod a+x ./travelcosts

echo "Removing old version..."
rm /usr/local/bin/travelcosts

echo "Replacing with new version..."
cp ./travelcosts /usr/local/bin

echo "All done!"
