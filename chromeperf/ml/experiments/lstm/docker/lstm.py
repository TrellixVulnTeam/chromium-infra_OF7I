# Copyright 2021 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

from scipy import stats
from keras.layers import Input, Dropout, Dense, LSTM, TimeDistributed, RepeatVector
import seaborn as sns
import joblib
from keras import regularizers
from keras.models import Model
from google.cloud import bigquery
import matplotlib.pyplot as plt
from sklearn.preprocessing import MinMaxScaler
from tensorflow.keras import layers
from tensorflow import keras
import tensorflow as tf
import pandas as pd
import numpy as np
import os
from datetime import datetime

sns.set(color_codes=True)

# See chromeperf/ml/experiments/lstm/doc/lstm.md for more information about
# approach and purpose


# Please note that these are initial values and approximations. No fine tuning
# has been conducted to fit the approach to our specific data and requirements # yet.
def autoencoder_model(X):
    inputs = Input(shape=(X.shape[1], X.shape[2]))
    L1 = LSTM(16,
              activation='relu',
              return_sequences=True,
              kernel_regularizer=regularizers.l2(0.00))(inputs)
    L2 = LSTM(4, activation='relu', return_sequences=False)(L1)
    L3 = RepeatVector(X.shape[1])(L2)
    L4 = LSTM(4, activation='relu', return_sequences=True)(L3)
    L5 = LSTM(16, activation='relu', return_sequences=True)(L4)
    output = TimeDistributed(Dense(X.shape[2]))(L5)
    model = Model(inputs=inputs, outputs=output)
    return model


bigquery_client = bigquery.Client(project='chromeperf-datalab')
#we can change this later to loop through different cases
# for multiple different combinations of metric, bot and platform
case = "ChromiumPerf", "linux-perf", "system_health.common_desktop/timeToFirstPaint/%/%"

sql = """
SELECT
        SPLIT(measurement, '/')[SAFE_OFFSET(2)] label,
        revision,
        value,
        std_error,
        sample_values
FROM `chromeperf.chromeperf_dashboard_data.rows`
WHERE DATE(timestamp) >= DATE_SUB(CURRENT_DATE(), INTERVAL 30 DAY)
AND master = "{MASTER}"
AND bot = "{BOT}"
AND measurement LIKE "{METRIC}"
ORDER BY revision ASC
"""


def fetch_sample_data(case):
    MASTER, BOT, METRIC = case
    return bigquery_client.query(
        sql.format(**locals())).result().to_dataframe()


df = fetch_sample_data(case)

# Data Preprocessing
# Here we are organising the data to be grouped by label and revision number.
label_group = df.groupby([
    'label',
    'revision',
])['sample_values'].apply(pd.Series.to_numpy)
groups = tuple(zip(*label_group.keys()))[0]

# This gets an array of the keys in the dictionary.
label_names = np.unique(groups)
label_stats = {}
for label in label_names:
    revisions_dictionary = label_group[label]
    revisions_list = list(revisions_dictionary.keys())
    stats_dictionary = {}
    for revision in revisions_list:
        values = np.concatenate(revisions_dictionary[revision], axis=0)
        median = np.median(values)
        average = np.average(values)
        std = np.std(values)
        mean = np.mean(values)
        minimum = np.min(values)
        maximum = np.max(values)
        delta = maximum - minimum
        IQR = stats.iqr(values, interpolation='midpoint')
        stats_dictionary[revision] = {
            'total': np.sum(values),
            'median': median,
            'average': average,
            'standard deviation': std,
            'mean': mean,
            'min': minimum,
            'max': maximum,
            'delta': delta,
            'IQR': IQR
        }
    label_stats[label] = stats_dictionary

# Model building and training
#
# We will create a model for each label so we must loop through each label in
# the data prepared above.
for label in label_stats:
    current_directory = os.getcwd()
    final_directory = os.path.join(current_directory, label)
    os.makedirs(final_directory)
    current_path = os.getcwd()

    # We are transposing the data in order to get it into a layout that the encoder will process.
    stats = pd.DataFrame.from_dict(label_stats[label]).transpose()

    # Here we are doing the training testing split for the data.
    rowcount = stats.shape[0]
    training = round(rowcount * 0.75)
    training_set = stats.iloc[0:training]  #is this an inclusive slice
    testing_set = stats.iloc[training:]

    # Raw data plots:
    # Training data.
    fig, ax = plt.subplots(figsize=(14, 6), dpi=80)
    ax.plot(training_set['median'],
            label='median',
            color='blue',
            animated=True,
            linewidth=1)
    ax.plot(training_set['min'],
            label='min',
            color='black',
            animated=True,
            linewidth=1)
    ax.plot(training_set['max'],
            label='max',
            color='red',
            animated=True,
            linewidth=1)
    ax.plot(training_set['delta'],
            label='delta',
            color='green',
            animated=True,
            linewidth=1)
    ax.plot(training_set['IQR'],
            label='IQR',
            color='magenta',
            animated=True,
            linewidth=1)
    plt.legend(loc='lower left')
    ax.set_title('Training data for ' + label, fontsize=14)
    my_file = label + '/training_' + label + '.png'
    path = os.path.join(current_path, my_file)
    fig.savefig(path)

    # Testing data.
    fig, ax = plt.subplots(figsize=(14, 6), dpi=80)
    ax.plot(testing_set['median'],
            label='median',
            color='blue',
            animated=True,
            linewidth=1)
    ax.plot(testing_set['min'],
            label='min',
            color='black',
            animated=True,
            linewidth=1)
    ax.plot(testing_set['max'],
            label='max',
            color='red',
            animated=True,
            linewidth=1)
    ax.plot(testing_set['delta'],
            label='delta',
            color='green',
            animated=True,
            linewidth=1)
    ax.plot(testing_set['IQR'],
            label='IQR',
            color='magenta',
            animated=True,
            linewidth=1)
    plt.legend(loc='lower left')
    ax.set_title('Testing data for ' + label, fontsize=14)
    my_file = label + '/testing_' + label + '.png'
    path = os.path.join(current_path, my_file)
    fig.savefig(path)

    # Fourier transform
    #
    # This transform is used to determine whether there are frequencies that
    # dominate the data. Any major changes are easily identified when studying
    # the frequency domain. We do not use this data, this is purely for context
    # for the reader.
    train_fft_set = np.fft.fft(training_set)
    test_fft_set = np.fft.fft(testing_set)

    # Here we plot the different training data for our model on the same axes.
    fig, ax = plt.subplots(figsize=(14, 6), dpi=80)
    ax.plot(train_fft_set[:, 1].real,
            label='median',
            color='blue',
            animated=True,
            linewidth=1)
    ax.plot(train_fft_set[:, 5].real,
            label='min',
            color='black',
            animated=True,
            linewidth=1)
    ax.plot(train_fft_set[:, 6].real,
            label='max',
            color='red',
            animated=True,
            linewidth=1)
    ax.plot(train_fft_set[:, 7].real,
            label='delta',
            color='green',
            animated=True,
            linewidth=1)
    ax.plot(train_fft_set[:, 8].real,
            label='IQR',
            color='magenta',
            animated=True,
            linewidth=1)
    plt.legend(loc='lower left')
    ax.set_title('Training data for ' + label, fontsize=14)
    my_file = label + '/fourierTraining_' + label + '.png'
    path = os.path.join(current_path, my_file)
    fig.savefig(path)

    # Here we plot the different test data for our model on the same axes.
    fig, ax = plt.subplots(figsize=(14, 6), dpi=80)
    ax.plot(test_fft_set[:, 1].real,
            label='median',
            color='blue',
            animated=True,
            linewidth=1)
    ax.plot(test_fft_set[:, 5].real,
            label='min',
            color='black',
            animated=True,
            linewidth=1)
    ax.plot(test_fft_set[:, 6].real,
            label='max',
            color='red',
            animated=True,
            linewidth=1)
    ax.plot(test_fft_set[:, 7].real,
            label='delta',
            color='green',
            animated=True,
            linewidth=1)
    ax.plot(test_fft_set[:, 8].real,
            label='IQR',
            color='magenta',
            animated=True,
            linewidth=1)
    plt.legend(loc='lower left')
    ax.set_title('Test data for ' + label, fontsize=14)
    my_file = label + '/fourierTest_' + label + '.png'
    path = os.path.join(current_path, my_file)
    fig.savefig(path)

    # To complete the pre-processing of our data, we will first normalize it to
    # a range between 0 and 1. Then we reshape our data so that it is in a
    # suitable format to be input into an LSTM network. LSTM cells expect a 3
    # dimensional tensor of the form [data samples,time/revision
    # steps,features]. Here, each sample input into the LSTM network represents
    # one revision (which acts as one step in time) and contains 5 features —
    # the statistics found for the collection of sample values at that revision.
    scaler = MinMaxScaler()
    X_train = scaler.fit_transform(training_set)
    X_test = scaler.transform(testing_set)
    scaler_filename = "scaler_data"
    joblib.dump(scaler, scaler_filename)

    # Here we are reshaping the training data so that it will be processed by
    # our autoencoder.
    X_train = X_train.reshape(X_train.shape[0], 1, X_train.shape[1])
    print("training data shape for " + label, X_train.shape)
    X_test = X_test.reshape(X_test.shape[0], 1, X_test.shape[1])
    print("test data shape for " + label, X_test.shape)

    # Here we are building the model.
    model = autoencoder_model(X_train)
    model.compile(optimizer='adam', loss='mae')
    model.summary()

    # We train the model over 100 epochs. An epoch is one cycle through the full
    # training dataset. So in this scenario, we are running the model over the
    # training set 100 times in order to complete the training.
    nb_epochs = 100
    batch_size = 10
    history = model.fit(X_train,
                        X_train,
                        epochs=nb_epochs,
                        batch_size=batch_size,
                        validation_split=0.05).history

    # We plot the losses found in training to evaluate the performance of the
    # model we have built.
    fig, ax = plt.subplots(figsize=(14, 6), dpi=80)
    ax.plot(history['loss'], 'b', label='Train', linewidth=2)
    ax.plot(history['val_loss'], 'y', label='Validation', linewidth=2)
    ax.set_title('Model loss for ' + label, fontsize=14)
    ax.set_ylabel('Loss (mae)')
    ax.set_xlabel('Epoch')
    ax.legend(loc='upper right')
    plt.show()

    # Here we see the loss distribution plot.
    X_pred = model.predict(X_train)
    X_pred = X_pred.reshape(X_pred.shape[0], X_pred.shape[2])
    X_pred = pd.DataFrame(X_pred, columns=training_set.columns)
    X_pred.index = training_set.index
    scored = pd.DataFrame(index=training_set.index)
    Xtrain = X_train.reshape(X_train.shape[0], X_train.shape[2])
    scored['Loss_mae'] = np.mean(np.abs(X_pred - Xtrain), axis=1)
    plt.figure(figsize=(16, 9), dpi=80)
    plt.title('Loss Distribution for ' + label, fontsize=16)
    sns.distplot(scored['Loss_mae'], bins=20, kde=True, color='blue')
    plt.xlim([0.0, .5])
    sum_stats = scored['Loss_mae'].describe(percentiles=[.9, .95, .99])

    # The tutorial followed in this notebook suggests that the way to determine
    # the threshold is by analysing the loss distribution graph to determine the
    # point at which the loss is negligible. However, we have decided to apply
    # the 99th percentile as the threshold as it seems to provide a more
    # reliable marker for acceptance.
    X_pred = model.predict(X_test)
    X_pred = X_pred.reshape(X_pred.shape[0], X_pred.shape[2])
    X_pred = pd.DataFrame(X_pred, columns=testing_set.columns)
    X_pred.index = testing_set.index
    scored = pd.DataFrame(index=testing_set.index)
    Xtest = X_test.reshape(X_test.shape[0], X_test.shape[2])
    scored['Loss_mae'] = np.mean(np.abs(X_pred - Xtest), axis=1)
    scored['Threshold'] = sum_stats['99%']
    scored['Anomaly'] = scored['Loss_mae'] > scored['Threshold']
    scored.plot(logy=True,
                figsize=(16, 9),
                ylim=[1e-2, 1e2],
                color=['blue', 'yellow'])
    label_dict = label_stats[label]
    label_dict['model'] = model

    # Here we are finally saving the model to our Google Cloud Storage bucket.
    # TODO(dberris): Maybe make this configurable?
    model.save("gs://chromeperf-datalab-kubeflow-experiments/lstm_models/" +
               label)
