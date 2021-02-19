# Copyright 2020 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

#TODO(dberris): move these to requirements.txt instead
#Required Installations
#!pip3 install --upgrade 'google-cloud-bigquery[pandas]' --user
#!pip3 install --upgrade pip --user
#!pip3 install scikit-learn
#!pip3 install google-cloud-bigquery-storage
#!pip3 install matplotlib
#!pip3 install kneed
#%load_ext google.cloud.bigquery

import os
import numpy
import pandas as pd
from google.cloud import bigquery
import matplotlib.pyplot as plt
from kneed import KneeLocator
from sklearn.datasets import make_blobs
from sklearn.cluster import KMeans
from sklearn.metrics import silhouette_score
from sklearn.preprocessing import StandardScaler
from sklearn import preprocessing

# See README at infra/chromeperf/ml/experiments/k_means/docs/k_means.md for more
# information about the k-means approach.

bigquery_client = bigquery.Client(project='chromeperf-datalab')
# can change this later to be arrays so we can loop through them for multiple
# different combinations of metric, bot and platform.
case = (
    'ChromiumPerf',
    'linux-perf',
    'system_health.common_desktop/timeToFirstPaint/%/%',
)

sql = """
SELECT SPLIT(measurement, '/')[SAFE_OFFSET(2)] label, revision, value, std_error,
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

# The query determines the data that we will be pulling from bigquery. For the
# POC we will be analysing the timeToFirstPaint metric over the last 30 days. We
# have pulled out labels from the measurement field.

# Data Preprocessing Here we are organising the data to be grouped by label and
# revision number.
label_group = df.groupby(['label', 'revision'
                          ])['sample_values'].apply(pd.Series.to_numpy)
groups = tuple(zip(*label_group.keys()))[0]
# get an array of the keys in the dictionary
label_names = numpy.unique(groups)

# This will create a dictionary of centroid clusters for each label for the
# current metric.
group_centroids = {}

for label in label_names:
    centroid_x = []
    centroid_y = []
    revisions_dictionary = label_group[label]
    revisions = list(revisions_dictionary.keys())

    window_size = 50
    # This is the initial window size approximation. Small window size allows
    # for enough data be studied at any one time without introducing additional
    # noise from larger datasets. This is an initial approximation and will need
    # to be tweaked further after testing to determine optimal sizings.
    window_number = 0
    i = 0
    window_centroids = {}

    # Here we will find the k-means clustering over each window.
    while i in range(0, len(revisions) - (window_size + 1)):
        selection = revisions[i:i + window_size]
        frames = []

        for revision_label in selection:
            sample_values = numpy.concatenate(
                revisions_dictionary[revision_label], axis=0)
            dataset = pd.DataFrame({
                'Revision': revision_label,
                'Values': sample_values[0:]
            })
            frames.append(dataset)
        result = pd.concat(frames)
        numpy_result = result.to_numpy()
        scaler = StandardScaler()
        scaled_features = scaler.fit_transform([(y, y)
                                                for _, y in numpy_result])
        kmeans = KMeans(init="random",
                        n_clusters=3,
                        n_init=10,
                        max_iter=300,
                        random_state=42)
        kmeans.fit(scaled_features)
        kmeans_kwargs = {
            "init": "random",
            "n_init": 10,
            "max_iter": 300,
            "random_state": 42,
        }

        # SILHOUTTE METHOD
        #
        # This list holds the silhouette coefficients for each k.
        silhouette_coefficients = []
        # We start at 2 clusters for silhouette coefficient.
        for k in range(2, 11):
            kmeans = KMeans(n_clusters=k, **kmeans_kwargs)
            kmeans.fit(scaled_features)
            score = silhouette_score(scaled_features, kmeans.labels_)
            silhouette_coefficients.append(score)
        # From the silhoutte method, we get the k_value.
        k_value = numpy.argmax(silhouette_coefficients) + 2
        kmeans_optimised = KMeans(n_clusters=k_value).fit(numpy_result)
        centers = numpy.array(kmeans_optimised.cluster_centers_)
        x, y = zip(*centers)
        centroid_x = centroid_x + ([window_number] * k_value)
        centroid_y = centroid_y + list(y)
        window_centroids[window_number] = list(y)
        i += (window_size - 10)
        window_number += 1
    group_centroids[label] = window_centroids

# Centroid plots for each label.
for key in group_centroids:
    label_dict = group_centroids[key].copy()
    x_axis = []
    y_axis = []
    for window in label_dict:
        values = label_dict[window]
        window_array = [window] * len(values)
        if len(x_axis) == 0:
            x_axis = window_array
            y_axis = values
        else:
            x_axis = x_axis + window_array
            y_axis = y_axis + values
    plt.figure()
    plt.scatter(x_axis, y_axis, color='black')
    plt.xlabel("window number")
    plt.ylabel("centroid values")
    plt.xticks(numpy.arange(0, max(x_axis) + 1, 1.0))
    plt.title(key)
    plt.savefig(key + "_centroid_plot.png")
