FROM alpine:latest

RUN apk add --update nodejs npm

WORKDIR /

COPY package*.json ./

RUN npm install

COPY . .

CMD ["npm", "run", "prod"]