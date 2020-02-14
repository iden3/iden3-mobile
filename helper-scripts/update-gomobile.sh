# TODO: add set -xe
echo "Binding the go code for Android" && \
cd ../go/mobile && gomobile bind --target android -o ./iden3-mobile.aar && jar xf iden3-mobile-sources.jar &&\
echo "Importing artifacts to Android project" && \
cp iden3-mobile.aar ../../android/app/libs/ && \
echo "Importing artifacts to Fluter project" && \
cp iden3-mobile.aar ../../flutter/android/app/src/main/libs/ && \
echo "Done!"
