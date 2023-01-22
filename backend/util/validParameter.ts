const validUsername = (username: string): boolean => {
    let regexp = new RegExp('^[a-zA-Z0-9_-]{3,12}$')
    return regexp.test(username)
}

const LimitParameter = (request: any) : boolean =>{
    return isNaN(request);
}

export {validUsername, LimitParameter}