FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
ENV IDENTITY_API_BASE=http://127.0.0.1:8080
ENV IDENTITY_ADMIN_TOKEN=build-placeholder
RUN npm run build
EXPOSE 3002
CMD ["npm", "run", "start"]
