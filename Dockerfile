FROM alpine:latest

RUN apk add --update nodejs npm

WORKDIR /app

COPY package*.json ./

RUN npm install

COPY . .

WORKDIR /app/frontend

RUN npm install

WORKDIR /app

CMD ["npm", "run", "prod"]