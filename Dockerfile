FROM node:18

WORKDIR /app

COPY package*.json ./

RUN npm install

COPY . .

WORKDIR /app/frontend

RUN npm install

WORKDIR /app