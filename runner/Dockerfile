FROM alpine:latest
WORKDIR /workspace
USER root
RUN apk add --no-cache libstdc++ cmake gcompat
RUN adduser -D app
RUN chown -R app:app /workspace
USER app
CMD ["sh"]