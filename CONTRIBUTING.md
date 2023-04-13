# Contributing to Logstash-exporter

## Guidelines

1. Create an issue: Before working on any changes, it's a good idea to create an issue to discuss whether the changes are actually needed and would benefit the project. This will help us understand your motivation and guide you in the right direction.

2. Fork the repository: Once the issue has been discussed and it's clear that your proposed changes are needed, fork the repository and make your changes on a separate branch.

3. Write tests: While 100% test coverage is not a strict requirement, I strongly encourage you to write tests for your changes. This will help ensure that your changes do not break existing functionality and will make the codebase more robust.

4. Pass CI: Your pull request must pass the continuous integration (CI) checks. If the CI checks fail, please fix any issues that are flagged and update your pull request.

5. Submit a pull request: Once your changes are complete and your branch is in sync with the main branch, submit a pull request. In the pull request description, please provide a brief summary of your changes, as well as any relevant information that reviewers should be aware of.

## Testing

### Changing application code

After changing application code, you can run the tests with:

   make test

It is recommended to track the code coverage of your changes. You can do this by running:

   make test-coverage

The aim is to have 100% code coverage. If you are unable to achieve this, please explain why in your pull request.

### Changing Helm chart

Testing the Helm chart is currently handled on CI using [kind](https://kind.sigs.k8s.io/). Chart testing process is described by code in [pipeline definition file](./.github/workflows/go-application.yml).
Currently there is no easy way to test the chart locally, but this is planned to be implemented in the future.

### Other changes

If you are making changes that do not affect the application code or the Helm chart, please explain how you tested your changes in your pull request.

## Code style

Please follow the existing code style in the project. This will make it easier for others to read and understand your changes.

## Commit messages

Write clear and concise commit messages that describe the changes you made and the motivation behind them. This will make it easier for others to review your changes and understand the history of the project. For previous commit messages, please see the commit history, for example on [GitHub repo history](https://github.com/kuskoman/logstash-exporter/commits/master).

## License

By contributing to this project, you agree that your contributions will be licensed under the [MIT License](./LICENSE).

## Special thanks

Thank you once again for considering contributing to Logstash-exporter! Your time and expertise are greatly appreciated.
