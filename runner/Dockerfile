FROM alpine:latest
WORKDIR /workspace
USER root
RUN apk add --no-cache libstdc++ cmake
RUN adduser -D app
USER app
CMD ["sh"]