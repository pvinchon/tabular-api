FROM node:22-slim

RUN npm install -g firebase-tools

RUN firebase setup:emulators:ui

COPY firebase.json .

HEALTHCHECK \
    --interval=5s \
    --timeout=5s \
    --retries=5 \
    CMD timeout 5s bash -c ':> /dev/tcp/127.0.0.1/4000' || exit 1

CMD ["sh", "-c", "firebase emulators:start --project $FIREBASE_PROJECT_ID"]

