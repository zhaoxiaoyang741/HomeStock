# syntax=docker/dockerfile:1
# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /app
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN corepack enable && pnpm build

# Stage 2: Nginx
FROM nginx:1.27-alpine

COPY --from=frontend-builder /app/dist/ /usr/share/nginx/html/
COPY deploy/docker/nginx/default.conf /etc/nginx/conf.d/default.conf
