FROM node:18-slim

WORKDIR /app

COPY package*.json ./

RUN npm install

COPY . .

WORKDIR /app/frontend

RUN npm install

WORKDIR /app

CMD ["npm", "run", "prod"]