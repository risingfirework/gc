# Hanya Chromium yang diperlukan. Memasang browser dari lockfile Playwright
# menghindari image full-browser yang jauh lebih besar (Chromium/Firefox/WebKit).
FROM node:22-bookworm-slim

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci --omit=dev --ignore-scripts
RUN npx playwright install --with-deps --only-shell chromium

COPY server.js ./

EXPOSE 8080
USER node
CMD ["node", "server.js"]
