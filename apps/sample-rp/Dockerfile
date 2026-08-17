FROM node:22-alpine
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm install
COPY . .
ENV OIDC_ISSUER=http://localhost:8080
ENV OIDC_CLIENT_ID=sample-rp
ENV OIDC_REDIRECT_URI=http://localhost:3001/callback
RUN npm run build
EXPOSE 3001
CMD ["npm", "run", "start"]
