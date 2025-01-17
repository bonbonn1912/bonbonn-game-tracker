import mongoose from 'mongoose';

const logSchema = new mongoose.Schema({
      text: {
        type: String,
        required: true,
      },
})
export default logSchema
// Export the model
;