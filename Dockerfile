FROM alpine:latest

RUN apk add --update nodejs npm

WORKDIR /

COPY package*.json ./

COPY . .

RUN npm install

CMD ["npm", "run", "prod"]