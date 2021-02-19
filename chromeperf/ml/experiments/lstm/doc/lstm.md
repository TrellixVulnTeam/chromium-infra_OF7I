# The approach
We approach the data using an unsupervised learning method, specifically LSTM
neural network with Autoencoder architecture, that is implemented in Python
using Keras.

We decided to use LSTM (i.e., Long Short Term Memory model), an artificial
recurrent neural network (RNN). This network is based on the basic structure of
RNNs, which are designed to handle sequential data, where the output from the
previous step is fed as input to the current step. LSTM is an improved version
of the vanilla RNN, and has three different “memory” gates: forget gate, input
gate and output gate. The forget gate controls what information in the cell
state to forget, given new information that entered from the input gate. Our
data is a time series (with revision numbers acting as the timestamps) so LSTM
is a good fit, thus, it was chosen as a promising approach.

# Neural Network Model
## LSTM autoencoder
We will use an autoencoder neural network architecture for this anomaly
detection model. The way this works is that the model takes the training data,
compresses it to its core features and then learns how to reconstruct that same
data. This way the model learns what is 'normal' data and makes assumptions on
how it will reconstruct the newer data coming in that it hasn't trained on. When
it encounters data that varies from the norm, it will have a higher level of
reconstruction error, thus identifying possible anomalous points.

The autoencoder is made up of 7 layers, including an input and an output layer.
The first two layers conduct the compressed representation of the data; the
encoder. The third layer, the repeat vector layer distributes this compressed
vector across the time steps of the decoder.

When we instantiate the model, we use 'Adam' as the optimiser of the neural
network and 'MAE' to calculate the loss function. Adam is a replacement
optimisation for stochastic gradient descent which combines AdaGrad and RMSProp
algorithms to create an algorithm that can handle sparse gradients on noisy
problems. Mae (Mean absolute error) is a loss function used for regression
models. It is the sum of absolute differences between the target and the
predicted variables. This means taht it measures the average magnitude of the
errors in a set of predictions without considering directions.

## Data description

The data we will be working with is currently being stored in the
chromeperf-datalab bigquery project.

The raw data is structured in the following way
- revision  INTEGER  --> This is the unique revision number for the build
- value FLOAT    --> This is generally the mean of sample_values
- std_error FLOAT    --> Standard error for value
- timestamp TIMESTAMP    --> This is the time the data was uploaded to the
  dashboard
- master    STRING --> Project
- bot   STRING --> Platform
- test  STRING -->Measurement data with the added information about the ----
  platform and the project --> Project/platform/measurement
- properties    STRING --> Properties of the test
- sample_values FLOAT -->An array of values from multiple iterations of the
  specific story.
- measurement STRING --> The exact metric being measured. The structure of the
  measurement usually is: *benchmark/metric/label/story*. Metrics have several
  label groups and each of those labels have individual stories to test.