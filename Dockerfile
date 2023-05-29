FROM scratch
ARG TARGETPLATFORM
COPY docker/$TARGETPLATFORM/tflcycles_exporter /
USER nobody
EXPOSE 9722
ENTRYPOINT ["/tflcycles_exporter"]
