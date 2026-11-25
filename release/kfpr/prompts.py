"""Terminal formatting and interactive prompts."""

import re

ANSI_BOLD = '\033[1m'


ANSI_UNDERLINE = '\033[4m'


ANSI_RESET = '\033[0m'


URL_RE = re.compile(r'https?://\S+')


def underline_links(text: str) -> str:
    """Underline URLs in terminal output."""
    return URL_RE.sub(
        lambda match: f'{ANSI_UNDERLINE}{match.group(0)}{ANSI_RESET}', text
    )


def emphasize_prompt(text: str) -> str:
    """Bold a prompt that needs user input."""
    return f'{ANSI_BOLD}{text}{ANSI_RESET}'


def prompt_choice(
    question: str, choices: list[str], default: str | None = None
) -> str:
    """Prompt user to choose from a list of options.

    Args:
      question: The question to ask.
      choices: List of valid choices.
      default: Default choice if user just presses enter.

    Returns:
      The selected choice.
    """
    suffix = f' [{default}]' if default else ''
    while True:
        answer = (
            input(
                emphasize_prompt(f'{question} ({"/".join(choices)}){suffix}: ')
            )
            .strip()
            .lower()
        )
        if not answer and default:
            return default
        if answer in choices:
            return answer
        print(f'Choose one of: {", ".join(choices)}')


def prompt_numbered_choice(question: str, choices: list[str]) -> str:
    """Prompt user to choose from a numbered list."""
    while True:
        print(emphasize_prompt(f'{question}:'))
        for index, choice in enumerate(choices, start=1):
            print(f'{index}) {choice}')
        answer = input(emphasize_prompt('Select number: ')).strip()
        if answer.isdigit() and 1 <= int(answer) <= len(choices):
            return choices[int(answer) - 1]
        print(f'Choose a number from 1 to {len(choices)}.')


def prompt_required(question: str) -> str:
    """Prompt user for required input.

    Args:
      question: The question to ask.

    Returns:
      The user input (non-empty).
    """
    while True:
        answer = input(emphasize_prompt(f'{question}: ')).strip()
        if answer:
            return answer
        print('A value is required.')


def prompt_validated(question: str, validator) -> str:
    """Prompt until validator accepts the answer."""
    while True:
        answer = prompt_required(question)
        try:
            validator(answer)
            return answer
        except ValueError as error:
            print(error)


def confirm(question: str) -> None:
    """Prompt user for confirmation.

    Args:
      question: The question to ask.

    Raises:
      SystemExit: If user does not confirm.
    """
    answer = prompt_choice(question, ['y', 'n'], default='n')
    if answer != 'y':
        raise SystemExit('Stopped by user.')
