import React from "react";
import Modal from "./Modal";
import MarkdownContent from "./MarkdownContent";
import { SlashUserCommand } from "../services/api";

interface CommandViewerModalProps {
  command: SlashUserCommand;
  onClose: () => void;
}

function CommandViewerModal({ command, onClose }: CommandViewerModalProps) {
  return (
    <Modal isOpen onClose={onClose} title={`/${command.name}`} className="skill-viewer-modal">
      <p className="skill-viewer-path" title={command.path}>{command.path}</p>
      {command.description && (
        <p className="command-viewer-description">{command.description}</p>
      )}
      {command.argument_hint && (
        <p className="command-viewer-hint"><strong>Argument hint:</strong> <code>{command.argument_hint}</code></p>
      )}
      <div className="skill-viewer-content">
        <MarkdownContent text={command.body} />
      </div>
    </Modal>
  );
}

export default CommandViewerModal;