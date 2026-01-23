FROM node:20-alpine AS base

WORKDIR /app
COPY package*.json ./
RUN npm install

FROM base AS dev
COPY . .
EXPOSE 3000
CMD ["npm", "run", "dev", "--", "--host"]

FROM base AS builder
COPY . .
RUN npm run build

FROM nginx:alpine AS prod
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
