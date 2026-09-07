# ¿Who Knows? - Flask application

A search engine from 2009 built with the latest technology! Python 3 and Flask 3.1.3. 

**Note**: This application is intentionally full of problems and vulnerabilities. Do not run it in a production environment. 

## Installation

All commands below are run from the `src` directory.

Create and activate a virtual environment so the dependencies do not conflict with
other Python projects:

```bash
$ uv venv
$ source .venv/bin/activate
```

Windows users activate with `.\.venv\Scripts\activate` instead.

Install the dependencies:

```bash
$ uv pip install -r backend/requirements.txt
```

`pip install -r backend/requirements.txt` works too if you are not using `uv`.

To initialize a new database:

```bash
$ make init
```

Note: Windows does not natively support Make. 


## Running the application

Start a development server on port `8080`:

```bash
$ make run
```
Or:

```bash
$ python3 ./backend/app.py
```

## Test the application

To run the tests:

```bash
$ make test
```

Or:

```bash
$ PYTHONPATH=backend python3 ./backend/app_tests.py
```
