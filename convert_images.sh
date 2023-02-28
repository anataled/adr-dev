#!/bin/bash

set -e

folders=("assets/image/jpg" "assets/image/png")

for f in ${folders[@]}; do 
    cd "$f"
    for i in *; do
        ffmpeg -y -i "$i" -c:v libwebp "../webp/${i%.*}.webp";
        ffmpeg -y -i "$i" -c:v libaom-av1 -still-picture 1 "../avif/${i%.*}.avif";
    done
    cd "../../.."
done