# syntax=docker/dockerfile:1.7
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

FROM node:22-alpine AS build
WORKDIR /app
ARG NEXT_PUBLIC_API_URL=/api/v1
ARG NEXT_PUBLIC_DEMO_EXAM_ID=30000000-0000-0000-0000-000000000001
ARG NEXT_PUBLIC_GOOGLE_CLIENT_ID=
ARG API_PROXY_TARGET=http://backend:8080
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_DEMO_EXAM_ID=$NEXT_PUBLIC_DEMO_EXAM_ID
ENV NEXT_PUBLIC_GOOGLE_CLIENT_ID=$NEXT_PUBLIC_GOOGLE_CLIENT_ID
ENV API_PROXY_TARGET=$API_PROXY_TARGET
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
ENV HOSTNAME=0.0.0.0
ENV PORT=3000
COPY --from=build --chown=node:node /app/public ./public
COPY --from=build --chown=node:node /app/.next/standalone ./
COPY --from=build --chown=node:node /app/.next/static ./.next/static
EXPOSE 3000
USER node
CMD ["node", "server.js"]
