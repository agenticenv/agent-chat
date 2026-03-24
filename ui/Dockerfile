# Build stage (context: ./ui)
FROM node:20-alpine AS builder

WORKDIR /app

COPY package.json package-lock.json* ./
RUN npm ci

COPY . .

RUN npm run build

# Run stage
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
# Override at runtime: docker run -e PORT=4000 -e API_BASE=http://backend:8080/api ...
ENV PORT=3000
ENV API_BASE=/api

# Copy built app and package files
COPY --from=builder /app/build ./build
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/package-lock.json* ./
COPY entrypoint.sh /entrypoint.sh

RUN npm ci --omit=dev && chmod +x /entrypoint.sh

EXPOSE 3000

ENTRYPOINT ["/entrypoint.sh"]
CMD ["npm", "start"]
