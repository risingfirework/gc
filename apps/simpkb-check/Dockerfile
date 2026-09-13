# Image resmi Playwright sudah berisi Chromium yang kompatibel dengan
# versi library playwright di package.json (1.48.2); heredity browser di
# /ms-playwright agar tidak mengunduh yang baru saat npm install.
FROM mcr.microsoft.com/playwright:v1.48.2-jammy

WORKDIR /app

COPY package.json ./
RUN npm install --omit=dev --ignore-scripts && \
    npx playwright install-deps chromium

COPY server.js ./

EXPOSE 8080
CMD ["node", "server.js"]