# G8 sandbox: runs the released static Linux binary on alpine.
# No host path is mounted on purpose. The only inputs are the binary, this
# image's platforms instance, and the driver script.
FROM node:22-alpine
RUN apk add --no-cache coreutils
WORKDIR /g8
COPY fenjue-agent-linux-amd64 /g8/bin/fenjue-agent
COPY platforms.g8.json /g8/platforms.json
COPY run_g8.sh /g8/run_g8.sh
RUN chmod +x /g8/bin/fenjue-agent /g8/run_g8.sh
ENTRYPOINT ["/bin/sh", "/g8/run_g8.sh"]