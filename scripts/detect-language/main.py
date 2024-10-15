import pandas as pd
import langid

try:
    import os
    os.remove("english_reviews_py.csv")
except:
    pass

with pd.read_csv('.data/reviews.csv', chunksize=1024) as reader:
    for reviews in reader:
        reviews_columns = ['app_id', 'review_text']
        reviews = reviews.filter(reviews_columns, axis="columns").dropna()
        reviews['review_text'] = reviews['review_text'].astype(str)
        reviews = reviews.head(100)

        def detect_language(texto):
            language, _ = langid.classify(texto)
            return language

        reviews["review_language"] = reviews['review_text'].apply(detect_language)
        reviews = reviews[reviews["review_language"] == "en"] # type: ignore
        reviews = reviews.filter(reviews_columns, axis="columns")

        reviews.to_csv("english_reviews_py.csv", mode='a', header=False, index=False)
